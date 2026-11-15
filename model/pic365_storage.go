package model

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const Pic365StorageCategory = "pic365_storage"
const Pic365StorageModel = "pic365-storage"
const Pic365StorageVersion = 1

var ErrPic365StorageAuthorization = errors.New("PIC365_STORAGE_AUTHORIZATION_REQUIRED")
var ErrPic365StorageConflict = errors.New("PIC365_STORAGE_INVOICE_CONFLICT")
var ErrPic365StorageLimit = errors.New("PIC365_STORAGE_MONTHLY_LIMIT")
var storageInvoicePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,64}$`)

// Storage mandates are separate from short-lived inference/session grants.
type Pic365StorageMandate struct {
	ID                string `gorm:"primaryKey;type:varchar(64)" json:"id"`
	CredentialHash    string `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	UserID            int    `gorm:"not null;index" json:"-"`
	Subject           string `gorm:"type:varchar(64);not null" json:"-"`
	AuthVersion       int64  `json:"-"`
	UnitPriceCents    int64  `json:"unit_price_cents_per_gib"`
	MonthlyLimitCents int64  `json:"monthly_limit_cents"`
	CreatedAt         int64  `json:"created_at"`
	ExpiresAt         int64  `json:"expires_at"`
	RevokedAt         int64  `json:"revoked_at"`
}

type Pic365StorageInvoice struct {
	ID             string  `gorm:"primaryKey;type:varchar(64)" json:"invoice_id"`
	MandateID      string  `gorm:"type:varchar(64);not null;index" json:"-"`
	UserID         int     `gorm:"not null;index:idx_pic_storage_user_month,priority:1" json:"-"`
	BillingMonth   string  `gorm:"type:varchar(7);index:idx_pic_storage_user_month,priority:2" json:"billing_month"`
	UsageDate      string  `gorm:"type:varchar(10)" json:"usage_date"`
	PeakBytes      int64   `json:"peak_bytes"`
	UnitPriceCents int64   `json:"unit_price_cents_per_gib"`
	TargetCents    int64   `json:"target_cents"`
	AmountCents    int64   `json:"amount_cents"`
	Quota          int     `json:"quota"`
	QuotaPerUSD    float64 `json:"quota_per_usd"`
	CNYPerUSD      float64 `json:"cny_per_usd"`
	BalanceBefore  int     `json:"balance_before"`
	BalanceAfter   int     `json:"balance_after"`
	CreatedAt      int64   `gorm:"index" json:"created_at"`
	LogPublished   bool    `json:"log_published"`
}

type Pic365StorageChargeRequest struct {
	InvoiceID      string `json:"invoice_id"`
	BillingMonth   string `json:"billing_month"`
	UsageDate      string `json:"usage_date"`
	PeakBytes      int64  `json:"peak_bytes"`
	UnitPriceCents int64  `json:"unit_price_cents_per_gib"`
	TargetCents    int64  `json:"target_cents"`
}

func storageCredentialHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func CreatePic365StorageMandate(token *Token, binding string, price, limit int64, days int) (*Pic365StorageMandate, string, error) {
	if price < 1 || price > 100000 || limit < 1 || limit > 10000000 || days < 1 || days > 365 || ValidatePic365Grant(token) != nil {
		return nil, "", ErrPic365StorageAuthorization
	}
	secret, err := common.GenerateKey()
	if err != nil {
		return nil, "", err
	}
	var mandate Pic365StorageMandate
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, token.UserId).Error; err != nil {
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return ErrPic365StorageAuthorization
		}
		currentBinding, err := getPic365GrantBindingWithTx(tx, token)
		if err != nil || subtle.ConstantTimeCompare([]byte(currentBinding), []byte(binding)) != 1 {
			return ErrPic365StorageAuthorization
		}
		now := common.GetTimestamp()
		// Changing consent replaces older mandates; their receipts remain queryable.
		if err := tx.Model(&Pic365StorageMandate{}).Where("user_id = ? AND revoked_at = 0", user.Id).Update("revoked_at", now).Error; err != nil {
			return err
		}
		mandate = Pic365StorageMandate{ID: uuid.NewString(), CredentialHash: storageCredentialHash(secret), UserID: user.Id,
			Subject: user.Pic365Subject, AuthVersion: user.AuthVersion, UnitPriceCents: price, MonthlyLimitCents: limit,
			CreatedAt: now, ExpiresAt: time.Now().AddDate(0, 0, days).Unix()}
		return tx.Create(&mandate).Error
	})
	return &mandate, secret, err
}

func GetPic365StorageMandate(secret string) (*Pic365StorageMandate, error) {
	if len(secret) < 32 || len(secret) > 128 {
		return nil, ErrPic365StorageAuthorization
	}
	var mandate Pic365StorageMandate
	if err := DB.Where("credential_hash = ?", storageCredentialHash(secret)).First(&mandate).Error; err != nil {
		return nil, ErrPic365StorageAuthorization
	}
	return &mandate, nil
}

func RevokePic365StorageMandate(secret string) error {
	mandate, err := GetPic365StorageMandate(secret)
	if err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, mandate.UserID).Error; err != nil {
			return err
		}
		return tx.Model(mandate).Where("revoked_at = 0").Update("revoked_at", common.GetTimestamp()).Error
	})
}

func ValidatePic365StorageMandate(mandate *Pic365StorageMandate, user *User) error {
	if mandate == nil || user == nil || mandate.UserID != user.Id || mandate.RevokedAt != 0 || mandate.ExpiresAt <= common.GetTimestamp() ||
		user.Status != common.UserStatusEnabled || user.Pic365Subject != mandate.Subject || user.AuthVersion != mandate.AuthVersion {
		return ErrPic365StorageAuthorization
	}
	return nil
}

func ChargePic365Storage(secret string, req Pic365StorageChargeRequest) (*Pic365StorageInvoice, error) {
	if !storageInvoicePattern.MatchString(req.InvoiceID) || req.PeakBytes < 0 || req.PeakBytes > 1<<50 || req.UnitPriceCents < 1 || req.UnitPriceCents > 100000 {
		return nil, ErrPic365StorageConflict
	}
	date, err := time.Parse("2006-01-02", req.UsageDate)
	if err != nil || date.Format("2006-01") != req.BillingMonth {
		return nil, ErrPic365StorageConflict
	}
	target := decimal.NewFromInt(req.PeakBytes).Mul(decimal.NewFromInt(req.UnitPriceCents)).Div(decimal.NewFromInt(1 << 30)).Floor().IntPart()
	if target != req.TargetCents || target < 1 || target > 10000000 {
		return nil, ErrPic365StorageConflict
	}
	mandate, err := GetPic365StorageMandate(secret)
	if err != nil {
		return nil, err
	}
	var invoice Pic365StorageInvoice
	newCharge := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, mandate.UserID).Error; err != nil {
			return ErrPic365StorageAuthorization
		}
		var stored Pic365StorageMandate
		if err := tx.First(&stored, "id = ?", mandate.ID).Error; err != nil {
			return ErrPic365StorageAuthorization
		}
		findErr := tx.First(&invoice, "id = ?", req.InvoiceID).Error
		if findErr == nil {
			if invoice.UserID != user.Id || invoice.MandateID != stored.ID || invoice.BillingMonth != req.BillingMonth || invoice.UsageDate != req.UsageDate ||
				invoice.PeakBytes != req.PeakBytes || invoice.UnitPriceCents != req.UnitPriceCents || invoice.TargetCents != req.TargetCents {
				return ErrPic365StorageConflict
			}
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		if err := ValidatePic365StorageMandate(&stored, &user); err != nil {
			return err
		}
		location := time.FixedZone("Asia/Shanghai", 8*3600)
		today := time.Now().In(location).Format("2006-01-02")
		start := time.Unix(stored.CreatedAt, 0).In(location).Format("2006-01-02")
		if req.UsageDate < start || req.UsageDate > today || req.UnitPriceCents != stored.UnitPriceCents {
			return ErrPic365StorageAuthorization
		}
		var previous int64
		var changedPrice int64
		if err := tx.Model(&Pic365StorageInvoice{}).Where("user_id = ? AND billing_month = ? AND unit_price_cents <> ?", user.Id, req.BillingMonth, req.UnitPriceCents).Count(&changedPrice).Error; err != nil {
			return err
		}
		if changedPrice != 0 {
			return ErrPic365StorageConflict
		}
		if err := tx.Model(&Pic365StorageInvoice{}).Where("user_id = ? AND billing_month = ?", user.Id, req.BillingMonth).Select("COALESCE(SUM(amount_cents), 0)").Scan(&previous).Error; err != nil {
			return err
		}
		amount := target - previous
		if amount <= 0 {
			return ErrPic365StorageConflict
		}
		monthStart, _ := time.ParseInLocation("2006-01-02", today[:7]+"-01", location)
		var monthSpent int64
		if err := tx.Model(&Pic365StorageInvoice{}).Where("user_id = ? AND created_at >= ?", user.Id, monthStart.Unix()).Select("COALESCE(SUM(amount_cents), 0)").Scan(&monthSpent).Error; err != nil {
			return err
		}
		if amount > stored.MonthlyLimitCents-monthSpent {
			return ErrPic365StorageLimit
		}
		rate, unit := operation_setting.USDExchangeRate, common.QuotaPerUnit
		if rate <= 0 || unit <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || math.IsNaN(unit) || math.IsInf(unit, 0) {
			return errors.New("PIC365_STORAGE_CURRENCY_INVALID")
		}
		quota, err := common.WalletQuotaFromDecimalStrict(decimal.NewFromInt(amount).Div(decimal.NewFromInt(100)).Div(decimal.NewFromFloat(rate)).Mul(decimal.NewFromFloat(unit)).Ceil())
		if err != nil || quota <= 0 {
			return errors.New("PIC365_STORAGE_CURRENCY_INVALID")
		}
		before := user.Quota
		allocations, err := consumeUserQuotaTx(tx, user.Id, quota)
		if err != nil {
			return err
		}
		paid, gift, _ := SumQuotaFundingAllocations(allocations)
		now := common.GetTimestamp()
		invoice = Pic365StorageInvoice{ID: req.InvoiceID, MandateID: stored.ID, UserID: user.Id, BillingMonth: req.BillingMonth,
			UsageDate: req.UsageDate, PeakBytes: req.PeakBytes, UnitPriceCents: req.UnitPriceCents, TargetCents: target, AmountCents: amount,
			Quota: quota, QuotaPerUSD: unit, CNYPerUSD: rate, BalanceBefore: before, BalanceAfter: before - quota, CreatedAt: now}
		if err := tx.Create(&invoice).Error; err != nil {
			return err
		}
		if err := createUserBalanceLedgerTx(tx, RecordUserBalanceLedgerParams{UserId: user.Id, OccurredAt: now,
			BucketType: "mixed", EntryType: CustomerMonthlyStatementEntryTypeConsume, AmountQuota: -quota, PaidQuota: -paid, GiftQuota: -gift,
			QuotaPerUnitSnapshot: unit, SourceType: Pic365StorageCategory, SourceName: "Pic365 storage", ExternalRef: invoice.ID,
			Remark: fmt.Sprintf("Pic365 storage | %s | peak %d bytes | CNY %.2f | invoice %s", invoice.BillingMonth, invoice.PeakBytes, float64(amount)/100, invoice.ID)}); err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error; err != nil {
			return err
		}
		newCharge = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if newCharge {
		syncFinanceQuotaDelta(invoice.UserID, -invoice.Quota)
	}
	if err := PublishPic365StorageLog(invoice.ID); err != nil {
		common.SysError("Pic365 storage usage log pending: " + invoice.ID)
	} else {
		invoice.LogPublished = true
	}
	return &invoice, nil
}

// The durable invoice is the outbox. Publishing never creates channel accounting.
func PublishPic365StorageLog(invoiceID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var invoice Pic365StorageInvoice
		if err := lockForUpdate(tx).First(&invoice, "id = ?", invoiceID).Error; err != nil {
			return err
		}
		if invoice.LogPublished {
			return nil
		}
		logDB := LOG_DB
		if DB == LOG_DB {
			logDB = tx
		}
		var count int64
		if err := logDB.Model(&Log{}).Where("request_id = ? AND user_id = ? AND model_name = ? AND type = ?", invoice.ID, invoice.UserID, Pic365StorageModel, LogTypeConsume).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			var user User
			if err := tx.Select("username").First(&user, invoice.UserID).Error; err != nil {
				return err
			}
			other := NewLogOther()
			other.SetPublic("billing_category", Pic365StorageCategory)
			other.SetPublic("billing_source", "wallet")
			other.SetPublic("storage_invoice", invoice)
			content := other.setContent([]*common.Message{common.NewMessage("Pic365 storage fee: {{month}}, CNY {{amount}}, invoice {{invoice}}", map[string]any{
				"month": invoice.BillingMonth, "amount": fmt.Sprintf("%.2f", float64(invoice.AmountCents)/100), "invoice": invoice.ID})})
			log := Log{UserId: invoice.UserID, Username: user.Username, Type: LogTypeConsume, CreatedAt: invoice.CreatedAt,
				ModelName: Pic365StorageModel, Quota: invoice.Quota, RequestId: invoice.ID, Content: content, Other: other.JSONString()}
			if err := logDB.Create(&log).Error; err != nil {
				return err
			}
		}
		return tx.Model(&invoice).Update("log_published", true).Error
	})
}

func GetPic365StorageReceipt(secret, invoiceID string) (*Pic365StorageInvoice, error) {
	mandate, err := GetPic365StorageMandate(secret)
	if err != nil {
		return nil, err
	}
	var invoice Pic365StorageInvoice
	if err := DB.Where("id = ? AND user_id = ? AND mandate_id = ?", invoiceID, mandate.UserID, mandate.ID).First(&invoice).Error; err != nil {
		return nil, err
	}
	if PublishPic365StorageLog(invoice.ID) == nil {
		invoice.LogPublished = true
	}
	return &invoice, nil
}

func RetryPic365StorageLogs() {
	var invoices []Pic365StorageInvoice
	if DB.Where("log_published = ?", false).Order("created_at").Limit(100).Find(&invoices).Error != nil {
		return
	}
	for _, invoice := range invoices {
		_ = PublishPic365StorageLog(invoice.ID)
	}
}
