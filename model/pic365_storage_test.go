package model

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPic365ReleaseSnapshotMigration(t *testing.T) {
	if os.Getenv("PIC365_RELEASE_MIGRATION") != "true" {
		t.Skip("production-snapshot rehearsal is not configured")
	}
	dsn, err := url.Parse(os.Getenv("SQL_DSN"))
	require.NoError(t, err)
	require.Equal(t, "/finance_test_pic365_release_20261115", dsn.Path)
	require.Empty(t, os.Getenv("LOG_SQL_DSN"))
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	defer conn.Close()
	queries := []string{
		`SELECT md5(string_agg(concat_ws('|',id,username,quota,paid_quota,gift_quota,used_quota,request_count,status,auth_version),',' ORDER BY id)) FROM users`,
		`SELECT md5(string_agg(row_to_json(c)::text,',' ORDER BY id)) FROM channels c`,
		`SELECT md5(string_agg(row_to_json(o)::text,',' ORDER BY key)) FROM options o`,
		`SELECT md5(string_agg(concat_ws('|',id,user_id,key,remain_quota,used_quota,status),',' ORDER BY id)) FROM tokens`,
		`SELECT md5(string_agg(concat_ws('|',id,user_id,amount_quota,source_type,external_ref),',' ORDER BY id)) FROM user_balance_ledgers`,
	}
	before := make([]string, len(queries))
	for i, query := range queries {
		require.NoError(t, db.Raw(query).Scan(&before[i]).Error)
	}
	common.IsMasterNode = true
	for range 2 {
		require.NoError(t, InitDB())
		require.NoError(t, InitLogDB())
		require.NoError(t, CloseDB())
	}
	for i, query := range queries {
		var after string
		require.NoError(t, db.Raw(query).Scan(&after).Error)
		assert.Equal(t, before[i], after, "snapshot data changed during migration: check %d", i)
	}
	assert.True(t, db.Migrator().HasTable(&Pic365Grant{}))
	assert.True(t, db.Migrator().HasTable(&Pic365StorageMandate{}))
	assert.True(t, db.Migrator().HasTable(&Pic365StorageInvoice{}))
	assert.True(t, db.Migrator().HasColumn(&UserBalanceLedger{}, "PaidQuota"))
	assert.True(t, db.Migrator().HasColumn(&UserBalanceLedger{}, "GiftQuota"))
	t.Log("Production snapshot migrated twice; user wallets, channels, pricing options, tokens and existing ledger were preserved")
}

func TestPic365StorageAccounting(t *testing.T) {
	for _, engine := range []string{"sqlite", "postgres", "mysql"} {
		t.Run(engine, func(t *testing.T) {
			var dialect gorm.Dialector = sqlite.Open(":memory:")
			if engine != "sqlite" {
				env := "FINANCE_TEST_PG_DSN"
				if engine == "mysql" {
					env = "FINANCE_TEST_MYSQL_DSN"
				}
				dsn := os.Getenv(env)
				if dsn == "" {
					t.Skip("isolated database is not configured")
				}
				require.True(t, strings.Contains(dsn, "127.0.0.1") && strings.Contains(dsn, "/finance_test_pic365"))
				if engine == "mysql" {
					dialect = mysql.Open(dsn)
				} else {
					dialect = postgres.Open(dsn)
				}
			}
			db, err := gorm.Open(dialect, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			require.NoError(t, err)
			conn, err := db.DB()
			require.NoError(t, err)
			conn.SetMaxOpenConns(1)
			oldDB, oldLog := DB, LOG_DB
			oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
			oldQuota, oldRate, oldLogging := common.QuotaPerUnit, operation_setting.USDExchangeRate, common.LogConsumeEnabled
			DB, LOG_DB = db, db
			common.SetDatabaseTypes(common.DatabaseType(engine), common.DatabaseType(engine))
			common.QuotaPerUnit, operation_setting.USDExchangeRate, common.LogConsumeEnabled = 500000, 7, false
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = oldDB, oldLog
				common.SetDatabaseTypes(oldMainType, oldLogType)
				common.QuotaPerUnit, operation_setting.USDExchangeRate, common.LogConsumeEnabled = oldQuota, oldRate, oldLogging
				initCol()
				require.NoError(t, conn.Close())
			})
			models := []any{&User{}, &Token{}, &UserSession{}, &AuthFlow{}, &Pic365Grant{}, &Pic365StorageMandate{}, &Pic365StorageInvoice{},
				&Log{}, &Channel{}, &TopUp{}, &Checkin{}, &Redemption{}, &UserQuotaFunding{}, &UserBalanceLedger{}, &FinancialAuditLog{}, &ChannelCostLedger{}, &ChannelCostAllocation{}, &CustomerMonthlyStatement{}, &CustomerMonthlyStatementItem{}}
			models = append(models, FinanceMonthlyModels()...)
			require.NoError(t, db.Migrator().DropTable(models...))
			for range 2 {
				require.NoError(t, db.AutoMigrate(models...))
			}
			// Simulate an old ledger without new allocation columns, preserving its money.
			require.NoError(t, db.Migrator().DropColumn(&UserBalanceLedger{}, "PaidQuota"))
			require.NoError(t, db.Migrator().DropColumn(&UserBalanceLedger{}, "GiftQuota"))
			require.NoError(t, db.Omit("PaidQuota", "GiftQuota").Create(&UserBalanceLedger{UserId: 99, AmountQuota: 12345, AmountUSD: 0.02469, BucketType: "paid", EntryType: "topup"}).Error)
			require.NoError(t, db.Migrator().DropTable(&Pic365StorageMandate{}, &Pic365StorageInvoice{}))
			for range 2 {
				require.NoError(t, db.AutoMigrate(models...))
			}
			var oldLedger UserBalanceLedger
			require.NoError(t, db.Where("user_id=99").First(&oldLedger).Error)
			assert.Equal(t, 12345, oldLedger.AmountQuota)
			assert.Equal(t, 0.02469, oldLedger.AmountUSD)
			assert.Zero(t, oldLedger.PaidQuota)
			var version string
			query := "SELECT version()"
			if engine == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database %s", version)
			user := User{Username: "storage-local", AffCode: "storage-local", Status: common.UserStatusEnabled, AuthVersion: 1, Quota: 1000000, PaidQuota: 900000, GiftQuota: 100000}
			require.NoError(t, db.Create(&user).Error)
			session := UserSession{SID: "storage-session", UserID: user.Id, Status: UserSessionStatusActive, Version: 1, UserAuthVersion: 1, ExpiresAt: time.Now().Add(time.Hour).Unix()}
			require.NoError(t, db.Create(&session).Error)
			var token *Token
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				var err error
				token, _, err = CreatePic365GrantWithTx(tx, AuthSessionIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1}, "gpt-image-2", session.ExpiresAt)
				return err
			}))
			binding, err := GetPic365GrantBinding(token)
			require.NoError(t, err)
			_, _, err = CreatePic365StorageMandate(token, "wrong", 300, 1000, 30)
			assert.ErrorIs(t, err, ErrPic365StorageAuthorization)
			mandate, secret, err := CreatePic365StorageMandate(token, binding, 300, 1000, 30)
			require.NoError(t, err)
			assert.NotEqual(t, secret, mandate.CredentialHash)
			date := time.Now().In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
			req := Pic365StorageChargeRequest{InvoiceID: "storage-invoice-0001", BillingMonth: date[:7], UsageDate: date, PeakBytes: 1 << 30, UnitPriceCents: 300, TargetCents: 300}
			invoice, err := ChargePic365Storage(secret, req)
			require.NoError(t, err)
			assert.Equal(t, 214286, invoice.Quota)
			assert.EqualValues(t, 300, invoice.AmountCents)
			assert.True(t, invoice.LogPublished)
			var saved User
			require.NoError(t, db.First(&saved, user.Id).Error)
			assert.Equal(t, 785714, saved.Quota)
			assert.Equal(t, 785714, saved.PaidQuota)
			assert.Zero(t, saved.GiftQuota)
			assert.Zero(t, saved.RequestCount)
			var ledger UserBalanceLedger
			require.NoError(t, db.Where("external_ref=?", req.InvoiceID).First(&ledger).Error)
			assert.Equal(t, -100000, ledger.GiftQuota)
			assert.Equal(t, -114286, ledger.PaidQuota)
			assert.Equal(t, Pic365StorageCategory, ledger.SourceType)
			// Concurrent duplicate delivery returns one receipt and one debit.
			if engine != "sqlite" {
				conn.SetMaxOpenConns(5)
			}
			var wg sync.WaitGroup
			retryErrors := make(chan error, 5)
			for range 5 {
				wg.Add(1)
				go func() { defer wg.Done(); _, err := ChargePic365Storage(secret, req); retryErrors <- err }()
			}
			wg.Wait()
			close(retryErrors)
			for err := range retryErrors {
				require.NoError(t, err)
			}
			conn.SetMaxOpenConns(1)
			var count int64
			require.NoError(t, db.Model(&UserBalanceLedger{}).Where("external_ref=?", req.InvoiceID).Count(&count).Error)
			assert.EqualValues(t, 1, count)
			require.NoError(t, db.Model(&Log{}).Where("request_id=?", req.InvoiceID).Count(&count).Error)
			assert.EqualValues(t, 1, count, "storage logs remain visible when inference logging is disabled")
			var summaryLog Log
			require.NoError(t, db.Where("request_id=?", req.InvoiceID).First(&summaryLog).Error)
			var summaryOther map[string]any
			require.NoError(t, common.UnmarshalJsonStr(summaryLog.Other, &summaryOther))
			summaryOther["storage_invoice"].(map[string]any)["credential"] = "must-not-appear-in-summary"
			summaryOther["storage_invoice"].(map[string]any)["user_id"] = 999
			encoded, err := common.Marshal(summaryOther)
			require.NoError(t, err)
			summaryLog.Other = string(encoded)
			SummarizeLogs([]*Log{&summaryLog})
			assert.True(t, summaryLog.Summary)
			assert.Contains(t, summaryLog.Other, `"billing_category":"pic365_storage"`)
			assert.Contains(t, summaryLog.Other, `"quota_per_usd":500000`)
			assert.Contains(t, summaryLog.Other, `"amount_cents":300`)
			assert.NotContains(t, summaryLog.Other, "credential")
			assert.NotContains(t, summaryLog.Other, "user_id")
			require.NoError(t, db.Model(&ChannelCostLedger{}).Count(&count).Error)
			assert.Zero(t, count)
			month := newFinanceBillingMonth(time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, financeBillingLocation), 7, 1000, 500000, time.Now())
			source, username, err := financeMonthlySourceItems(month, user.Id)
			require.NoError(t, err)
			storageItems := []CustomerMonthlyStatementItem{}
			for _, item := range source {
				if item.OperationType == Pic365StorageCategory {
					storageItems = append(storageItems, item)
				}
			}
			require.Len(t, storageItems, 1)
			assert.Equal(t, -214286, storageItems[0].QuotaRaw)
			bill, items := buildFinanceMonthlyBill(month, user.Id, username, source, time.Now())
			assert.InDelta(t, float64(invoice.Quota)/500000, bill.ConsumeUSD, 0.00000001)
			csv, err := financeBillCSV(bill, items, false)
			require.NoError(t, err)
			assert.Contains(t, string(csv), req.InvoiceID)
			assert.Contains(t, string(csv), "Pic365")
			usage := financeBuildUserUsageFromStatementItems(storageItems)
			assert.Zero(t, usage[user.Id].ConsumeCount)
			bad := req
			bad.PeakBytes++
			bad.InvoiceID = "storage-invoice-bad01"
			_, err = ChargePic365Storage(secret, bad)
			assert.ErrorIs(t, err, ErrPic365StorageConflict)
			bad = req
			bad.InvoiceID = "storage-invoice-limit"
			bad.PeakBytes = 4 << 30
			bad.TargetCents = 1200
			_, err = ChargePic365Storage(secret, bad)
			assert.ErrorIs(t, err, ErrPic365StorageLimit)
			bad = req
			bad.InvoiceID = "storage-invoice-rollback"
			bad.PeakBytes = 2 << 30
			bad.TargetCents = 600
			// A failure after the quota update must roll back money and the receipt together.
			func() {
				require.NoError(t, db.Callback().Create().Before("gorm:create").Register("pic365:test_fail_invoice", func(tx *gorm.DB) {
					if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "Pic365StorageInvoice" {
						tx.AddError(errors.New("storage test invoice failure"))
					}
				}))
				defer db.Callback().Create().Remove("pic365:test_fail_invoice")
				_, err = ChargePic365Storage(secret, bad)
				require.ErrorContains(t, err, "storage test invoice failure")
			}()
			require.NoError(t, db.First(&saved, user.Id).Error)
			assert.Equal(t, 785714, saved.Quota)
			assert.Equal(t, 785714, saved.PaidQuota)
			assert.Zero(t, saved.GiftQuota)
			require.NoError(t, db.Model(&Pic365StorageInvoice{}).Where("id=?", bad.InvoiceID).Count(&count).Error)
			assert.Zero(t, count)
			require.NoError(t, db.Model(&UserBalanceLedger{}).Where("external_ref=?", bad.InvoiceID).Count(&count).Error)
			assert.Zero(t, count)
			require.NoError(t, db.Model(&user).Updates(map[string]any{"quota": 0, "paid_quota": 0, "gift_quota": 0}).Error)
			_, err = ChargePic365Storage(secret, bad)
			assert.ErrorIs(t, err, ErrFinanceInsufficientQuota)
			require.NoError(t, db.First(&saved, user.Id).Error)
			assert.Zero(t, saved.Quota)
			require.NoError(t, db.Model(&user).Updates(map[string]any{"quota": 785714, "paid_quota": 785714}).Error)
			require.NoError(t, db.Model(mandate).Update("expires_at", time.Now().Add(-time.Minute).Unix()).Error)
			_, err = ChargePic365Storage(secret, bad)
			assert.ErrorIs(t, err, ErrPic365StorageAuthorization)
			require.NoError(t, db.Model(mandate).Update("expires_at", time.Now().AddDate(0, 0, 30).Unix()).Error)
			// Disabled users, changed auth versions and invalid dates cannot create debt.
			for _, field := range []string{"status", "auth_version"} {
				value := int64(2)
				if field == "status" {
					value = int64(common.UserStatusDisabled)
				}
				require.NoError(t, db.Model(&user).Update(field, value).Error)
				bad = req
				bad.InvoiceID = "storage-invoice-auth01"
				bad.PeakBytes = 2 << 30
				bad.TargetCents = 600
				_, err = ChargePic365Storage(secret, bad)
				assert.ErrorIs(t, err, ErrPic365StorageAuthorization)
				require.NoError(t, db.Model(&user).Update(field, 1).Error)
			}
			require.NoError(t, db.Model(&UserSession{}).Where("sid=?", session.SID).Update("revoked_at", time.Now().Unix()).Error)
			bad = req
			bad.InvoiceID = "storage-invoice-durable"
			bad.PeakBytes = 2 << 30
			bad.TargetCents = 600
			func() {
				missingLogDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
				require.NoError(t, err)
				missingLogConn, err := missingLogDB.DB()
				require.NoError(t, err)
				LOG_DB = missingLogDB
				common.SetDatabaseTypes(common.DatabaseType(engine), common.DatabaseTypeSQLite)
				defer func() {
					LOG_DB = db
					common.SetDatabaseTypes(common.DatabaseType(engine), common.DatabaseType(engine))
					require.NoError(t, missingLogConn.Close())
				}()
				pending, err := ChargePic365Storage(secret, bad)
				require.NoError(t, err, "log failure cannot undo a committed debit")
				assert.False(t, pending.LogPublished)
				replayed, err := ChargePic365Storage(secret, bad)
				require.NoError(t, err, "storage consent outlives the login session")
				assert.Equal(t, pending.Quota, replayed.Quota)
			}()
			RetryPic365StorageLogs()
			RetryPic365StorageLogs()
			require.NoError(t, db.Model(&Log{}).Where("request_id=?", bad.InvoiceID).Count(&count).Error)
			assert.EqualValues(t, 1, count, "the receipt outbox publishes exactly once after log recovery")
			require.NoError(t, db.Model(&UserBalanceLedger{}).Where("external_ref=?", bad.InvoiceID).Count(&count).Error)
			assert.EqualValues(t, 1, count)
			_, _, err = CreatePic365StorageMandate(token, binding, 300, 1000, 30)
			assert.ErrorIs(t, err, ErrPic365StorageAuthorization)
			require.NoError(t, RevokePic365StorageMandate(secret))
			bad = req
			bad.InvoiceID = "storage-invoice-revoked"
			bad.PeakBytes = 3 << 30
			bad.TargetCents = 900
			_, err = ChargePic365Storage(secret, bad)
			assert.ErrorIs(t, err, ErrPic365StorageAuthorization)
			_, err = ChargePic365Storage(secret, req)
			require.NoError(t, err, "already-paid replay remains queryable after revocation")
			require.NoError(t, db.Where("request_id=?", req.InvoiceID).Delete(&Log{}).Error)
			receipt, err := GetPic365StorageReceipt(secret, req.InvoiceID)
			require.NoError(t, err)
			assert.Equal(t, invoice.Quota, receipt.Quota, "usage-log cleanup never removes the receipt")
			require.NoError(t, db.First(&saved, user.Id).Error)
			assert.Equal(t, 571428, saved.Quota)
			assert.Zero(t, saved.RequestCount)
			t.Log(fmt.Sprintf("storage receipt, ledger, monthly export and retry verified: %s", engine))
		})
	}
}
