package model

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"math"
	"strings"
	"unicode/utf8"
)

func (redemption *Redemption) SetFinanceFaceValue(currency string, amount float64) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency != "CNY" && currency != "USD" {
		return common.NewMessage("Unsupported top-up currency")
	}
	rate := operation_setting.USDExchangeRate
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) ||
		rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) ||
		common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return common.NewMessage("Invalid financial amount")
	}
	usd := decimal.NewFromFloat(amount)
	if currency == "CNY" {
		usd = usd.Div(decimal.NewFromFloat(rate))
	}
	quota, err := common.WalletQuotaFromDecimalStrict(usd.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0))
	if err != nil || quota <= 0 {
		return ErrInvalidTopUpQuota
	}
	redemption.Currency, redemption.InputAmount, redemption.ExchangeRate = currency, amount, rate
	redemption.Quota, redemption.QuotaPerUnitSnapshot = quota, common.QuotaPerUnit
	redemption.AmountUSD = quotaToUSDWithSnapshot(quota, common.QuotaPerUnit)
	return nil
}

func normalizeRedemptionFundingType(fundingType string) string {
	if strings.TrimSpace(fundingType) == QuotaFundingTypePaid {
		return QuotaFundingTypePaid
	}
	return QuotaFundingTypeGift
}

func quotaFromUSDWithSnapshot(amountUSD float64, quotaPerUnit float64) int {
	if amountUSD <= 0 || quotaPerUnit <= 0 {
		return 0
	}
	if math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) || math.IsNaN(quotaPerUnit) || math.IsInf(quotaPerUnit, 0) {
		return 0
	}
	quota, err := common.WalletQuotaFromDecimalStrict(decimal.NewFromFloat(amountUSD).Mul(decimal.NewFromFloat(quotaPerUnit)).Round(0))
	if err != nil {
		return 0
	}
	return quota
}

func (redemption *Redemption) ValidateFinance() error {
	if redemption.FundingType != "" && redemption.FundingType != QuotaFundingTypePaid && redemption.FundingType != QuotaFundingTypeGift {
		return common.NewMessage("Invalid funding type")
	}
	for _, value := range []float64{redemption.AmountUSD, redemption.RecognizedRevenueUSD, redemption.QuotaPerUnitSnapshot} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return common.NewMessage("Invalid financial amount")
		}
	}
	redemption.normalizeForPersistence()
	if redemption.FundingType == QuotaFundingTypeGift && redemption.Remark == "" {
		return common.NewMessage("Remark is required for gift vouchers")
	}
	if utf8.RuneCountInString(redemption.Remark) > 255 {
		return common.NewMessage("Remark must be 255 characters or fewer")
	}
	if redemption.Name == "" {
		label := "Free"
		if redemption.FundingType == QuotaFundingTypePaid {
			label = "Paid"
		}
		redemption.Name = fmt.Sprintf("%s $%.2f", label, redemption.AmountUSD)
	}
	if utf8.RuneCountInString(redemption.Name) > 20 {
		return common.NewMessage("Redemption code name length must be between 1-20")
	}
	if redemption.Quota <= 0 {
		return ErrInvalidTopUpQuota
	}
	return common.ValidateWalletQuota(redemption.Quota)
}

func normalizeRedemptionMoneyValue(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return roundAccountingAmount(value)
}

func (redemption *Redemption) normalizeForPersistence() {
	if redemption == nil {
		return
	}
	redemption.Name = strings.TrimSpace(redemption.Name)
	redemption.Remark = strings.TrimSpace(redemption.Remark)
	redemption.FundingType = normalizeRedemptionFundingType(redemption.FundingType)
	if redemption.QuotaPerUnitSnapshot <= 0 {
		redemption.QuotaPerUnitSnapshot = common.QuotaPerUnit
	}
	// The credited quota is the authoritative face value, including legacy codes.
	if redemption.Quota > 0 {
		redemption.AmountUSD = quotaToUSDWithSnapshot(redemption.Quota, redemption.QuotaPerUnitSnapshot)
	}
	if redemption.Quota <= 0 && redemption.AmountUSD > 0 {
		redemption.Quota = quotaFromUSDWithSnapshot(redemption.AmountUSD, redemption.QuotaPerUnitSnapshot)
	}
	redemption.AmountUSD = normalizeRedemptionMoneyValue(redemption.AmountUSD)
	if redemption.FundingType == QuotaFundingTypePaid {
		if redemption.RecognizedRevenueUSD <= 0 && redemption.AmountUSD > 0 {
			redemption.RecognizedRevenueUSD = redemption.AmountUSD
		}
		redemption.RecognizedRevenueUSD = normalizeRedemptionMoneyValue(redemption.RecognizedRevenueUSD)
		return
	}
	redemption.RecognizedRevenueUSD = 0
}

func (redemption *Redemption) normalizeForResponse() {
	if redemption == nil {
		return
	}
	redemption.FundingType = normalizeRedemptionFundingType(redemption.FundingType)
	if redemption.QuotaPerUnitSnapshot <= 0 {
		redemption.QuotaPerUnitSnapshot = common.QuotaPerUnit
	}
	if redemption.AmountUSD <= 0 && redemption.Quota > 0 {
		redemption.AmountUSD = quotaToUSDWithSnapshot(redemption.Quota, redemption.QuotaPerUnitSnapshot)
	}
	redemption.AmountUSD = normalizeRedemptionMoneyValue(redemption.AmountUSD)
	if redemption.FundingType == QuotaFundingTypePaid {
		if redemption.RecognizedRevenueUSD <= 0 && redemption.AmountUSD > 0 {
			redemption.RecognizedRevenueUSD = redemption.AmountUSD
		}
		redemption.RecognizedRevenueUSD = normalizeRedemptionMoneyValue(redemption.RecognizedRevenueUSD)
		return
	}
	redemption.RecognizedRevenueUSD = 0
}

func buildRedemptionGrantRemark(redemption *Redemption) string {
	if redemption == nil {
		return "redeemed voucher"
	}
	baseRemark := "redeemed free voucher"
	if normalizeRedemptionFundingType(redemption.FundingType) == QuotaFundingTypePaid {
		baseRemark = "redeemed paid voucher"
	}
	if redemption.Remark == "" {
		return baseRemark
	}
	return fmt.Sprintf("%s: %s", baseRemark, redemption.Remark)
}

func (redemption *Redemption) grantSourceType() string {
	if redemption != nil && normalizeRedemptionFundingType(redemption.FundingType) == QuotaFundingTypePaid {
		return QuotaFundingSourcePaidVoucher
	}
	return QuotaFundingSourceGiftVoucher
}

func normalizeRedemptionSlice(redemptions []*Redemption) {
	for _, redemption := range redemptions {
		redemption.normalizeForResponse()
	}
}
