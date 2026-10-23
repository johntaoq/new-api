package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrInvalidUserQuotaAdjustment = errors.New("invalid user quota adjustment")
	ErrUserQuotaPermission        = errors.New("cannot adjust quota for this user role")
)

// UserQuotaAdjustment is the immutable database snapshot of a committed manual
// adjustment. Pending relay deductions in the quota cache are not part of it.
type UserQuotaAdjustment struct {
	UserID   int
	Username string
	Before   int
	After    int
}

func AdjustUserQuota(userID, operatorRole int, mode string, value int, operators ...FinancialAuditOperator) (*UserQuotaAdjustment, error) {
	if userID <= 0 || (mode != "add" && mode != "subtract" && mode != "override") {
		return nil, ErrInvalidUserQuotaAdjustment
	}
	if mode != "override" && value <= 0 {
		return nil, ErrInvalidUserQuotaAdjustment
	}
	if value > common.MaxWalletQuota || value < -common.MaxWalletQuota {
		return nil, ErrWalletQuotaLimitExceeded
	}

	var adjustment UserQuotaAdjustment
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
			return err
		}
		if operatorRole != common.RoleRootUser && operatorRole <= user.Role {
			return ErrUserQuotaPermission
		}
		if user.Quota > common.MaxWalletQuota || user.Quota < -common.MaxWalletQuota {
			return ErrWalletQuotaLimitExceeded
		}
		if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
			return err
		}
		before := buildFinancialAuditUserQuotaSnapshot(user)
		operator := FinancialAuditOperator{}
		if len(operators) > 0 {
			operator = operators[0]
		}
		quota := decimal.NewFromInt(int64(value))
		switch mode {
		case "add":
			quota = decimal.NewFromInt(int64(user.Quota)).Add(quota)
		case "subtract":
			quota = decimal.NewFromInt(int64(user.Quota)).Sub(quota)
		}
		after, err := common.WalletQuotaFromDecimalStrict(quota)
		if err != nil {
			return ErrWalletQuotaLimitExceeded
		}
		delta := after - user.Quota
		if delta > 0 {
			for remaining := delta; remaining > 0; {
				amount := min(remaining, common.MaxWalletQuota)
				if _, err := grantUserQuotaTx(tx, QuotaFundingGrantParams{
					UserId: userID, FundingType: QuotaFundingTypeGift,
					SourceType: QuotaFundingSourceAdminGrant, GrantedQuota: amount,
					OperatorUserId: operator.UserId, OperatorUsernameSnapshot: operator.UsernameSnapshot,
					BalanceEntryType: CustomerMonthlyStatementEntryTypeAdjustment,
				}); err != nil {
					return err
				}
				remaining -= amount
			}
		} else if delta < 0 {
			for remaining := -delta; remaining > 0; {
				amount := min(remaining, common.MaxWalletQuota)
				allocations, err := consumeUserQuotaTx(tx, userID, amount, true)
				if err != nil {
					return err
				}
				paid, gift, _ := SumQuotaFundingAllocations(allocations)
				for bucket, quota := range map[string]int{QuotaFundingTypePaid: paid, QuotaFundingTypeGift: gift} {
					if err := createUserBalanceLedgerTx(tx, RecordUserBalanceLedgerParams{
						UserId: userID, BucketType: bucket, AmountQuota: -quota,
						EntryType:  CustomerMonthlyStatementEntryTypeAdjustment,
						SourceType: QuotaFundingSourceSystemAdjust, OperatorUserId: operator.UserId,
						OperatorUsernameSnapshot: operator.UsernameSnapshot,
					}); err != nil {
						return err
					}
				}
				remaining -= amount
			}
		}
		adjustment = UserQuotaAdjustment{UserID: user.Id, Username: user.Username, Before: user.Quota, After: after}
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module: FinancialAuditModuleUserQuota, Action: FinancialAuditActionAdjust,
			OperatorUserId: operator.UserId, OperatorUsernameSnapshot: operator.UsernameSnapshot,
			TargetType: FinancialAuditTargetUserWallet, TargetId: userID, TargetUserId: userID,
			Before: before, After: buildFinancialAuditUserQuotaSnapshot(user),
		})
	})
	if err != nil {
		return nil, err
	}

	// Apply only the committed difference, preserving outstanding reservations.
	// Both balances are bounded above, so their difference fits in int64.
	delta := int64(adjustment.After) - int64(adjustment.Before)
	if delta != 0 {
		if err := cacheIncrUserQuota(userID, delta); err != nil {
			common.SysError(common.LogText("failed to sync manual quota adjustment for user %d: %s", userID, err))
		}
	}
	return &adjustment, nil
}
