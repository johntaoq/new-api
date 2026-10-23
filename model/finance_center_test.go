package model

import (
	"crypto/sha256"
	"database/sql"
	csvreader "encoding/csv"
	"fmt"
	"math"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestFinanceAccountingAndStatements(t *testing.T) {
	for _, engine := range []string{"sqlite", "postgres", "mysql"} {
		t.Run(engine, func(t *testing.T) {
			var driver gorm.Dialector = sqlite.Open(":memory:")
			if engine == "postgres" {
				dsn := os.Getenv("FINANCE_TEST_PG_DSN")
				if dsn == "" {
					t.Skip("FINANCE_TEST_PG_DSN is not configured")
				}
				location, err := url.Parse(dsn)
				require.NoError(t, err)
				require.Equal(t, "127.0.0.1", location.Hostname(), "finance tests only use an isolated local database")
				require.True(t, strings.HasPrefix(location.Path, "/finance_test_"))
				driver = postgres.Open(dsn)
			}
			if engine == "mysql" {
				dsn := os.Getenv("FINANCE_TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("FINANCE_TEST_MYSQL_DSN is not configured")
				}
				config, err := mysqldriver.ParseDSN(dsn)
				require.NoError(t, err)
				require.True(t, strings.HasPrefix(config.Addr, "127.0.0.1:"))
				require.True(t, strings.HasPrefix(config.DBName, "finance_test_"))
				driver = mysql.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			oldDB, oldLogDB := DB, LOG_DB
			oldMain, oldLog := common.MainDatabaseType(), common.LogDatabaseType()
			oldQuota, oldLogging := common.QuotaPerUnit, common.LogConsumeEnabled
			DB, LOG_DB = db, db
			common.QuotaPerUnit = 500000
			common.SetDatabaseTypes(common.DatabaseType(engine), common.DatabaseType(engine))
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = oldDB, oldLogDB
				common.QuotaPerUnit, common.LogConsumeEnabled = oldQuota, oldLogging
				common.SetDatabaseTypes(oldMain, oldLog)
				initCol()
				require.NoError(t, sqlDB.Close())
			})
			for range 2 {
				require.NoError(t, db.AutoMigrate(FinanceMonthlyModels()...))
				require.NoError(t, db.AutoMigrate(
					&User{}, &Token{}, &Channel{}, &Log{}, &TopUp{}, &Redemption{}, &Checkin{},
					&UserQuotaFunding{}, &UserBalanceLedger{}, &FinancialAuditLog{},
					&ChannelCostLedger{}, &ChannelCostAllocation{},
					&CustomerMonthlyStatement{}, &CustomerMonthlyStatementItem{},
					&ChannelMonthlyStatement{}, &ChannelMonthlyStatementItem{},
				))
			}
			var version string
			versionSQL := "select sqlite_version()"
			if engine != "sqlite" {
				versionSQL = "select version()"
			}
			require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
			t.Log(version)
			t.Run("legacy_topup_migration", func(t *testing.T) {
				// Match the released finance columns, including their storage precision.
				const table = "finance_test_legacy_topups"
				require.NoError(t, db.Exec("CREATE TABLE "+table+" ("+
					"id bigint PRIMARY KEY, input_amount double precision NOT NULL DEFAULT 0, "+
					"currency varchar(10) NOT NULL DEFAULT '', exchange_rate double precision NOT NULL DEFAULT 0, "+
					"quota bigint NOT NULL DEFAULT 0, revenue_usd double precision NOT NULL DEFAULT 0)").Error)
				t.Cleanup(func() { require.NoError(t, db.Migrator().DropTable(table)) })
				order := TopUp{Id: 1, InputAmount: 10, Currency: "CNY", ExchangeRate: 7.123456789,
					Quota: 701906, RevenueUSD: 1.403812973}
				require.NoError(t, db.Exec("INSERT INTO "+table+
					" (id,input_amount,currency,exchange_rate,quota,revenue_usd) VALUES (?,?,?,?,?,?)",
					order.Id, order.InputAmount, order.Currency, order.ExchangeRate, order.Quota, order.RevenueUSD).Error)
				for range 2 {
					require.NoError(t, db.Table(table).AutoMigrate(&TopUp{}))
					var saved TopUp
					require.NoError(t, db.Table(table).First(&saved, order.Id).Error)
					assert.Equal(t, order.InputAmount, saved.InputAmount)
					assert.Equal(t, order.ExchangeRate, saved.ExchangeRate)
					assert.Equal(t, order.RevenueUSD, saved.RevenueUSD)
					assert.Equal(t, order.Quota, saved.Quota)
					assert.Equal(t, order.Currency, saved.Currency)
					columns, err := db.Migrator().ColumnTypes(table)
					require.NoError(t, err)
					for _, col := range columns {
						switch col.Name() {
						case "input_amount", "exchange_rate", "revenue_usd", "quota", "currency":
							nullable, ok := col.Nullable()
							require.True(t, ok)
							assert.False(t, nullable, col.Name())
							if engine == "postgres" && col.Name() != "quota" && col.Name() != "currency" {
								assert.Equal(t, "float8", col.DatabaseTypeName(), col.Name())
							}
						}
					}
				}
			})
			t.Run("legacy_wallet_statement_migration", func(t *testing.T) {
				const table = "finance_monthly_bills"
				require.NoError(t, db.Migrator().DropColumn(&FinanceMonthlyBill{}, "wallet_summary_json"))
				for _, column := range []string{"bucket_type", "billing_source", "remark"} {
					require.NoError(t, db.Migrator().DropColumn(&FinanceMonthlyBillItem{}, column))
				}
				require.NoError(t, db.Exec("INSERT INTO "+table+" (id,bill_month,user_id,consume_usd,consume_cny) VALUES (?,?,?,?,?)", 99, "2026-04", 99, 1.23456789, 8.64197523).Error)
				require.NoError(t, db.Exec("INSERT INTO finance_monthly_bill_items (id,bill_id,amount_usd,amount_cny) VALUES (?,?,?,?)", 99, 99, -1.23456789, -8.64197523).Error)
				legacyFile := FinanceMonthlyBillFile{BillId: 99, Content: []byte("original fixed statement\r\n")}
				legacyFile.SHA256 = fmt.Sprintf("%x", sha256.Sum256(legacyFile.Content))
				require.NoError(t, db.Create(&legacyFile).Error)
				for range 2 {
					require.NoError(t, db.AutoMigrate(FinanceMonthlyModels()...))
					var bill FinanceMonthlyBill
					require.NoError(t, db.Table(table).First(&bill, 99).Error)
					assert.Equal(t, 1.23456789, bill.ConsumeUSD)
					assert.Equal(t, 8.64197523, bill.ConsumeCNY)
					assert.Empty(t, bill.WalletSummaryJSON, "legacy statements are not silently recalculated")
					var item FinanceMonthlyBillItem
					require.NoError(t, db.First(&item, 99).Error)
					assert.Equal(t, -1.23456789, item.AmountUSD)
					assert.Equal(t, -8.64197523, item.AmountCNY)
					var file FinanceMonthlyBillFile
					require.NoError(t, db.First(&file, "bill_id = ?", 99).Error)
					assert.Equal(t, legacyFile.Content, file.Content)
					assert.Equal(t, legacyFile.SHA256, file.SHA256)
				}
				require.NoError(t, db.Delete(&FinanceMonthlyBillFile{}, "bill_id = ?", 99).Error)
				require.NoError(t, db.Delete(&FinanceMonthlyBillItem{}, 99).Error)
				require.NoError(t, db.Delete(&FinanceMonthlyBill{}, 99).Error)
			})
			user := User{Username: "=finance-test", Status: common.UserStatusEnabled}
			require.NoError(t, db.Create(&user).Error)
			t.Cleanup(func() {
				for _, table := range []any{
					&FinanceMonthlyBillFile{}, &FinanceMonthlyBillItem{}, &FinanceMonthlyBill{}, &FinanceBillingMonth{},
					&ChannelCostAllocation{}, &ChannelCostLedger{}, &CustomerMonthlyStatementItem{}, &CustomerMonthlyStatement{},
					&ChannelMonthlyStatementItem{}, &ChannelMonthlyStatement{}, &UserQuotaFunding{}, &UserBalanceLedger{},
					&FinancialAuditLog{}, &Log{}, &Token{}, &Channel{}, &TopUp{}, &Redemption{}, &Checkin{}, &User{},
				} {
					require.NoError(t, db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
				}
			})
			require.NoError(t, GrantPaidQuota(user.Id, 500000, QuotaFundingSourceStripe, 1, "local-test", 1, ""))
			require.NoError(t, GrantGiftQuota(user.Id, 100000, QuotaFundingSourceAdminGrant, 0, "", ""))
			allocations, err := ConsumeUserQuotaWithAllocation(user.Id, 200000)
			require.NoError(t, err)
			require.Len(t, allocations, 2)
			assert.Equal(t, QuotaFundingTypeGift, allocations[0].FundingType)
			assert.Equal(t, 100000, allocations[0].AllocatedQuota)
			paid, gift, revenue := SumQuotaFundingAllocations(allocations)
			assert.Equal(t, 100000, paid)
			assert.Equal(t, 100000, gift)
			assert.InDelta(t, 0.2, revenue, 0.000001)
			remaining, refunded, err := RefundUserQuotaAllocations(user.Id, allocations, 50000)
			require.NoError(t, err)
			require.Len(t, remaining, 2)
			require.NoError(t, db.First(&user, user.Id).Error)
			assert.Equal(t, 450000, user.Quota)
			assert.Equal(t, 450000, user.PaidQuota)
			assert.Zero(t, user.GiftQuota)

			month := time.Now().Format("2006-01")
			for i, event := range []struct {
				kind  string
				quota int
			}{{ChannelCostEntryTypeConsume, 200000}, {ChannelCostEntryTypeRefund, 50000}} {
				eventAllocations := allocations
				if i == 1 {
					eventAllocations = refunded
				}
				require.NoError(t, RecordChannelCostLedger(RecordChannelCostLedgerParams{
					UserId: user.Id, RequestId: "finance-lifecycle", EntryType: event.kind,
					BillingSource: "wallet", OriginModelName: "local-model", ActualQuota: event.quota,
					EstimatedCostUSD: float64(event.quota) / 1000000, Allocations: eventAllocations,
				}))
			}
			page := &common.PageInfo{Page: 1, PageSize: 10}
			beforeGeneration, err := ListFinanceCustomerBillSummaryV2(month, user.Username, page)
			require.NoError(t, err)
			assert.Zero(t, beforeGeneration.Total, "reading a report cannot generate snapshots")
			generated, err := GenerateFinanceCustomerBills(month, user.Username)
			require.NoError(t, err)
			assert.Equal(t, 1, generated.GeneratedUserCount)
			result, err := ListFinanceCustomerBillSummaryV2(month, user.Username, page)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.InDelta(t, 0.3, result.Summary.TotalConsumeUSD, 0.000001)
			assert.InDelta(t, 0.1, result.Summary.PaidConsumeUSD, 0.000001)
			assert.InDelta(t, 0.2, result.Summary.GiftConsumeUSD, 0.000001)
			csv, _, err := ExportFinanceCustomerBillCSVV2(month, user.Id)
			require.NoError(t, err)
			assert.Contains(t, string(csv), "'=finance-test")
			details, err := ListFinanceCustomerBillDetailsV2(month, user.Id, &common.PageInfo{Page: 1, PageSize: 1})
			require.NoError(t, err)
			assert.Equal(t, 4, details.Total)
			assert.Len(t, details.Items, 1)
			summary, err := BuildFinanceDashboardSummary("month", month)
			require.NoError(t, err)
			assert.InDelta(t, 1, summary.KPIs.SalesIncome.USD, 0.000001)
			channelSummary, err := BuildFinanceChannelCostSummary("month", month)
			require.NoError(t, err)
			assert.InDelta(t, 0.15, channelSummary.KPIs.TotalCostUSD, 0.000001)
			assert.InDelta(t, channelSummary.KPIs.TotalCostUSD, summary.KPIs.ChannelCost.USD, 0.000001,
				"dashboard cost must not use the customer's billed consumption")
			t.Run("model_costs_across_channels", func(t *testing.T) {
				period, err := parseFinancePeriod("month", "2001-02")
				require.NoError(t, err)
				rows := []ChannelCostLedger{
					{ChannelId: 101, ChannelNameSnapshot: "First", OriginModelName: "shared-model", EntryType: ChannelCostEntryTypeConsume,
						PromptTokens: 100, CompletionTokens: 20, CacheTokens: 10, TotalTokens: 120, RecognizedRevenueUSD: 4.123456, EstimatedCostUSD: 1.123456, OccurredAt: period.Start},
					{ChannelId: 102, ChannelNameSnapshot: "Second", OriginModelName: "shared-model", EntryType: ChannelCostEntryTypeConsume,
						PromptTokens: 200, CompletionTokens: 40, CacheTokens: 20, TotalTokens: 240, RecognizedRevenueUSD: 5, EstimatedCostUSD: 2, OccurredAt: period.End},
					{ChannelId: 102, ChannelNameSnapshot: "Second", OriginModelName: "shared-model", EntryType: ChannelCostEntryTypeRefund,
						PromptTokens: 10, CompletionTokens: 2, CacheTokens: 1, TotalTokens: 12, RecognizedRevenueUSD: 0.5, EstimatedCostUSD: 0.1, OccurredAt: period.Start + 1},
					{ChannelId: 101, ChannelNameSnapshot: "First", OriginModelName: "another-model", EntryType: ChannelCostEntryTypeConsume,
						EstimatedCostUSD: 0.5, OccurredAt: period.Start},
				}
				require.NoError(t, db.Create(&rows).Error)
				result, err := ListFinanceChannelCostModels("month", "2001-02", &common.PageInfo{Page: 1, PageSize: 1})
				require.NoError(t, err)
				assert.Equal(t, 2, result.Total)
				items := result.Items.([]FinanceChannelCostModelItem)
				require.Len(t, items, 1)
				assert.Equal(t, "shared-model", items[0].ModelName)
				assert.Equal(t, 290, items[0].PromptTokens)
				assert.Equal(t, 58, items[0].CompletionTokens)
				assert.Equal(t, 29, items[0].CacheTokens)
				assert.Equal(t, 348, items[0].TotalTokens)
				assert.InDelta(t, 8.623456, items[0].RevenueUSD, 0.0000001)
				assert.InDelta(t, 3.023456, items[0].CostUSD, 0.0000001, "aggregation must retain accounting precision")
				channels, err := ListFinanceChannelCostChannels("month", "2001-02", &common.PageInfo{Page: 1, PageSize: 10})
				require.NoError(t, err)
				assert.Equal(t, 2, channels.Total)
				t.Run("complete_csv_exports", func(t *testing.T) {
					for i := range 12 {
						require.NoError(t, db.Create(&ChannelCostLedger{ChannelId: 200 + i, ChannelNameSnapshot: fmt.Sprintf("channel-%02d", i),
							OriginModelName: fmt.Sprintf("model-%02d", i), EntryType: ChannelCostEntryTypeConsume, OccurredAt: period.Start,
							EstimatedCostUSD: 0.01}).Error)
					}
					require.NoError(t, db.Create(&ChannelCostLedger{ChannelId: 300, ChannelNameSnapshot: "=unsafe,\"channel\"",
						OriginModelName: "@unsafe,\"model\"", EntryType: ChannelCostEntryTypeRefund, OccurredAt: period.Start,
						EstimatedCostUSD: 0.25}).Error)
					for _, view := range []string{"channels", "models"} {
						content, filename, err := ExportFinanceChannelCostCSV("month", "2001-02", view)
						require.NoError(t, err)
						assert.Equal(t, view+"-costs-month-2001-02-USD.csv", filename)
						assert.True(t, strings.HasPrefix(string(content), "\ufeff"))
						assert.Contains(t, string(content), "\r\n")
						records, err := csvreader.NewReader(strings.NewReader(strings.TrimPrefix(string(content), "\ufeff"))).ReadAll()
						require.NoError(t, err)
						require.Len(t, records, 16, "download must include every row, not just the first ten")
						if view == "models" {
							assert.Equal(t, "模型", records[0][0])
							assert.NotContains(t, records[0], "渠道")
							assert.Equal(t, []string{"shared-model", "290.00", "58.00", "29.00", "348.00", "8.62", "3.02"}, records[1])
							assert.Equal(t, "'@unsafe,\"model\"", records[15][0])
						} else {
							assert.Equal(t, []string{"渠道", "供应商"}, records[0][:2])
							assert.Equal(t, "'=unsafe,\"channel\"", records[15][0])
						}
						assert.Equal(t, "-0.25", records[15][len(records[15])-1])
						for i := 3; i < 15; i++ {
							assert.Equal(t, fmt.Sprintf("%s-%02d", strings.TrimSuffix(view, "s"), i-3), records[i][0], "tied rankings have stable pagination")
						}
						yearContent, _, err := ExportFinanceChannelCostCSV("year", "2001", view)
						require.NoError(t, err)
						assert.Equal(t, content, yearContent)
						empty, _, err := ExportFinanceChannelCostCSV("month", "2000-01", view)
						require.NoError(t, err)
						emptyRecords, err := csvreader.NewReader(strings.NewReader(strings.TrimPrefix(string(empty), "\ufeff"))).ReadAll()
						require.NoError(t, err)
						assert.Len(t, emptyRecords, 1)
					}
					_, _, err := ExportFinanceChannelCostCSV("month", "2001-02", "invalid")
					require.Error(t, err)
					_, _, err = ExportFinanceChannelCostCSV("month", "not-a-month", "models")
					require.Error(t, err)
				})
			})
			t.Run("legacy_model_costs_across_channels", func(t *testing.T) {
				period, err := parseFinancePeriod("month", "2001-03")
				require.NoError(t, err)
				logs := []Log{
					{Type: LogTypeConsume, ChannelId: 401, ModelName: "legacy-shared", PromptTokens: 10, Quota: 500000, CreatedAt: period.Start},
					{Type: LogTypeConsume, ChannelId: 402, ModelName: "legacy-shared", PromptTokens: 20, Quota: 500000, CreatedAt: period.Start},
				}
				require.NoError(t, db.Create(&logs).Error)
				result, err := ListFinanceChannelCostModels("month", "2001-03", &common.PageInfo{Page: 1, PageSize: 10})
				require.NoError(t, err)
				assert.Equal(t, 1, result.Total)
				items := result.Items.([]FinanceChannelCostModelItem)
				require.Len(t, items, 1)
				assert.Equal(t, 30, items[0].PromptTokens)
				assert.Equal(t, "legacy-shared", items[0].ModelName)
			})

			_, err = ConsumeUserQuotaWithAllocation(user.Id, 450001)
			require.ErrorIs(t, err, ErrFinanceInsufficientQuota)
			debt, err := ConsumeUserQuotaWithAllocation(user.Id, 500000, true)
			require.NoError(t, err)
			require.NoError(t, db.First(&user, user.Id).Error)
			assert.Equal(t, -50000, user.Quota)
			_, _, err = RefundUserQuotaAllocations(user.Id, debt, 500000)
			require.NoError(t, err)
			require.NoError(t, db.First(&user, user.Id).Error)
			assert.Equal(t, 450000, user.Quota)
			_, _, err = RefundUserQuotaAllocations(user.Id, debt, 500000)
			require.Error(t, err, "a fully restored funding source cannot be refunded again")
			require.ErrorIs(t, GrantPaidQuota(user.Id, common.MaxWalletQuota, "test", 0, "", 1, ""), ErrTopUpQuotaLimitExceeded)
			require.Error(t, GrantPaidQuota(user.Id, 1, "test", 0, "", math.NaN(), ""))

			order := TopUp{UserId: user.Id, Amount: 1, Money: 7, TradeNo: "finance-test-order",
				PaymentMethod: "test", PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusPending}
			require.NoError(t, db.Create(&order).Error)
			done, err := RechargeEpay(order.TradeNo, "test", "")
			require.NoError(t, err)
			assert.False(t, done)
			done, err = RechargeEpay(order.TradeNo, "test", "")
			require.NoError(t, err)
			assert.True(t, done)
			var fundedCount int64
			require.NoError(t, db.Model(&UserQuotaFunding{}).Where("source_type = ? AND source_ref_id = ?", QuotaFundingSourceOnlineTopUp, order.Id).Count(&fundedCount).Error)
			assert.EqualValues(t, 1, fundedCount)

			keys, err := CreateRedemptionsWithAudit(Redemption{Name: "local voucher", Quota: 50000, FundingType: QuotaFundingTypePaid},
				1, FinancialAuditOperator{UserId: user.Id, UsernameSnapshot: user.Username})
			require.NoError(t, err)
			require.Len(t, keys, 1)
			var audit FinancialAuditLog
			require.NoError(t, db.Last(&audit).Error)
			assert.NotContains(t, audit.BeforeJSON+audit.AfterJSON, keys[0])
			_, err = Redeem(keys[0], user.Id)
			require.NoError(t, err)
			_, err = Redeem(keys[0], user.Id)
			require.Error(t, err)
			_, err = AdjustUserQuota(user.Id, common.RoleRootUser, "add", 50000, FinancialAuditOperator{UserId: user.Id})
			require.NoError(t, err)
			audit = FinancialAuditLog{}
			require.NoError(t, db.Last(&audit).Error)
			assert.Equal(t, FinancialAuditModuleUserQuota, audit.Module)
			var voucher Redemption
			require.NoError(t, db.Where(commonKeyCol+" = ?", keys[0]).First(&voucher).Error)
			deleted, err := BatchDeleteRedemptionsWithAudit([]int{voucher.Id, voucher.Id}, FinancialAuditOperator{UserId: user.Id})
			require.NoError(t, err)
			assert.EqualValues(t, 1, deleted)
			audit = FinancialAuditLog{}
			require.NoError(t, db.Last(&audit).Error)
			assert.Equal(t, FinancialAuditTargetRedemptionBatch, audit.TargetType)
			assert.NotContains(t, audit.BeforeJSON+audit.AfterJSON, keys[0])

			common.LogConsumeEnabled = false
			var logCount, ledgerCount int64
			require.NoError(t, db.Model(&Log{}).Count(&logCount).Error)
			require.NoError(t, db.Model(&ChannelCostLedger{}).Count(&ledgerCount).Error)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/", nil)
			RecordConsumeLog(ctx, user.Id, RecordConsumeLogParams{Quota: 100, Other: NewLogOther()})
			var afterLogs, afterLedgers int64
			require.NoError(t, db.Model(&Log{}).Count(&afterLogs).Error)
			require.NoError(t, db.Model(&ChannelCostLedger{}).Count(&afterLedgers).Error)
			assert.Equal(t, logCount, afterLogs)
			assert.Equal(t, ledgerCount+1, afterLedgers)
			t.Run("fixed_monthly_billing", func(t *testing.T) {
				testFixedMonthlyBilling(t, db, user.Id)
			})
			t.Run("wallet_statement_reconciliation", func(t *testing.T) {
				start := time.Date(2026, 5, 1, 0, 0, 0, 0, financeBillingLocation)
				account := User{Username: "statement-reconciliation", AffCode: "statement-reconciliation", Quota: 4950000, PaidQuota: 4700000, GiftQuota: 250000}
				require.NoError(t, db.Create(&account).Error)
				month := newFinanceBillingMonth(start, 7, 1000, 500000, start)
				for _, balance := range []UserBalanceLedger{
					{UserId: account.Id, EntryType: "topup", BucketType: "paid", AmountQuota: 5000000, AmountUSD: 8, QuotaPerUnitSnapshot: 500000, OccurredAt: start.Add(time.Hour).Unix(), CreatedAt: start.Add(time.Hour).Unix(), SourceType: "manual_topup", SourceRefId: 2},
					{UserId: account.Id, EntryType: "gift", BucketType: "gift", AmountQuota: 500000, QuotaPerUnitSnapshot: 500000, OccurredAt: start.Add(2 * time.Hour).Unix(), CreatedAt: start.Add(2 * time.Hour).Unix()},
					{UserId: account.Id, EntryType: "adjustment", BucketType: "paid", AmountQuota: -50000, QuotaPerUnitSnapshot: 500000, OccurredAt: start.Add(3 * time.Hour).Unix(), CreatedAt: start.Add(3 * time.Hour).Unix()},
				} {
					require.NoError(t, db.Create(&balance).Error)
				}
				require.NoError(t, db.Create(&UserQuotaFunding{UserId: account.Id, FundingType: "paid", GrantedQuota: 5000000, SourceType: "manual_topup", SourceRefId: 2, QuotaPerUnitSnapshot: 500000, CreatedAt: start.Add(time.Hour).Unix()}).Error)
				logs := []Log{
					{UserId: account.Id, Type: LogTypeConsume, Quota: 1000000, RequestId: "statement-wallet-consume", CreatedAt: start.Add(4 * time.Hour).Unix()},
					{UserId: account.Id, Type: LogTypeRefund, Quota: 500000, RequestId: "statement-wallet-refund", CreatedAt: start.Add(5 * time.Hour).Unix()},
					{UserId: account.Id, Type: LogTypeConsume, Quota: 9000000, RequestId: "statement-subscription", Other: `{"billing_source":"subscription"}`, CreatedAt: start.Add(6 * time.Hour).Unix()},
				}
				require.NoError(t, db.Create(&logs).Error)
				require.NoError(t, db.Create(&ChannelCostLedger{UserId: account.Id, LogId: logs[0].Id, RequestId: logs[0].RequestId, EntryType: "consume", ActualQuota: 1000000, InternalEquivalentUSD: 99, PaidQuotaUsed: 750000, GiftQuotaUsed: 250000, OccurredAt: logs[0].CreatedAt, CreatedAt: logs[0].CreatedAt}).Error)
				source, username, err := financeMonthlySourceItems(month, account.Id)
				require.NoError(t, err)
				require.Len(t, source, 6, "matched funding and consume logs are not duplicated")
				bill, items := buildFinanceMonthlyBill(month, account.Id, username, source, start.AddDate(0, 1, 0))
				require.NoError(t, attachFinanceWalletBalances(&bill, month, source, start.AddDate(0, 1, 0)))
				assert.Equal(t, 70.0, bill.WalletSummary.Topup.CNY, "wallet quota is not replaced by recognized revenue")
				assert.Equal(t, 7.0, bill.WalletSummary.Gift.CNY)
				assert.Equal(t, -0.7, bill.WalletSummary.Adjustment.CNY)
				assert.Equal(t, 14.0, bill.WalletSummary.Consumption.CNY)
				assert.Equal(t, 7.0, bill.WalletSummary.Refund.CNY)
				assert.Equal(t, 84.0, bill.WalletSummary.FundsIn.CNY, "total inflows include returned consumption")
				assert.Equal(t, 14.7, bill.WalletSummary.FundsOut.CNY, "total outflows include consumption and wallet reductions")
				assert.Equal(t, 1.0, bill.ConsumeUSD, "subscription usage does not debit the wallet")
				assert.Equal(t, 0.0, bill.WalletSummary.Opening.CNY)
				assert.Equal(t, 69.3, bill.WalletSummary.Closing.CNY)
				assert.Zero(t, bill.WalletSummary.Reconciliation.CNY)
				assert.Equal(t, "topup", items[0].EntryType)
				export, err := financeBillCSV(bill, items, false)
				require.NoError(t, err)
				assert.Contains(t, string(export), "上期余额(CNY)")
				assert.Contains(t, string(export), "70.00")
				assert.Less(t, strings.Index(string(export), "资金流水"), strings.Index(string(export), "API消费明细"))
				assert.NotContains(t, string(export), "Token数")
				for _, checkpoint := range []FinancialAuditLog{
					{Module: FinancialAuditModuleUserQuota, TargetType: FinancialAuditTargetUserWallet, TargetUserId: account.Id, CreatedAt: start.Add(time.Hour).Unix(), BeforeJSON: `{"quota":0}`, AfterJSON: `{"quota":5000000}`},
					{Module: FinancialAuditModuleUserQuota, TargetType: FinancialAuditTargetUserWallet, TargetUserId: account.Id, CreatedAt: month.EndsAt, BeforeJSON: `{"quota":5000000}`, AfterJSON: `{"quota":5000000}`},
				} {
					require.NoError(t, db.Create(&checkpoint).Error)
				}
				require.NoError(t, attachFinanceWalletBalances(&bill, month, source, start.AddDate(0, 1, 0)))
				assert.Zero(t, bill.WalletSummary.Opening.CNY)
				assert.Equal(t, 70.0, bill.WalletSummary.Closing.CNY)
				assert.InDelta(t, 0.7, bill.WalletSummary.Reconciliation.CNY, 0.00000001, "independent audit discrepancies are exposed, not hidden as adjustments")
				month.FinalizedAt = month.ClosesAt
				require.NoError(t, db.Create(&month).Error)
				require.NoError(t, db.Create(&bill).Error)
				next := newFinanceBillingMonth(start.AddDate(0, 1, 0), 8, 1000, 500000, start)
				nextBill, _ := buildFinanceMonthlyBill(next, account.Id, account.Username, nil, start.AddDate(0, 2, 0))
				require.NoError(t, attachFinanceWalletBalances(&nextBill, next, nil, start.AddDate(0, 2, 0)))
				assert.Equal(t, bill.WalletSummary.Closing, nextBill.WalletSummary.Opening)
				assert.Equal(t, "previous_fixed_statement", nextBill.WalletSummary.OpeningBasis)
				assert.Equal(t, 10.0, nextBill.WalletSummary.ExchangeChangeCNY)
				assert.Equal(t, 79.2, nextBill.WalletSummary.Closing.CNY)
				assert.Equal(t, -0.8, nextBill.WalletSummary.Reconciliation.CNY, "a newer live balance is not overwritten by an older audit")
				reconstructed, basis, err := financeWalletBalanceAt(account.Id, next.StartsAt, 500000, start.AddDate(0, 6, 0))
				require.NoError(t, err)
				assert.Equal(t, 10.0, reconstructed.InexactFloat64(), "historical boundaries prefer the nearest audit over a distant live balance")
				assert.Equal(t, "wallet_audit_reconstruction", basis)
				liveNow := start.AddDate(0, 2, 0).Add(time.Hour)
				current, basis, err := financeWalletBalanceAt(account.Id, liveNow.Unix()+1, 500000, liveNow)
				require.NoError(t, err)
				assert.Equal(t, 9.9, current.InexactFloat64(), "live reports use the current wallet, not an older audit checkpoint")
				assert.Equal(t, "current_wallet_snapshot", basis)
				export, err = financeBillCSV(bill, items, false)
				require.NoError(t, err)
				assert.Contains(t, string(export), "存在待核对差额")
			})
			t.Run("wallet_parity", func(t *testing.T) {
				snapshotRate := operation_setting.USDExchangeRate
				operation_setting.USDExchangeRate = 7
				t.Cleanup(func() { operation_setting.USDExchangeRate = snapshotRate })
				operator := FinancialAuditOperator{UserId: user.Id, UsernameSnapshot: user.Username}
				_, err := CreateRedemptionsWithAudit(Redemption{AmountUSD: 2, FundingType: QuotaFundingTypeGift}, 1, operator)
				require.Error(t, err, "gift vouchers require a reason like production")
				keys, err := CreateRedemptionsWithAudit(Redemption{AmountUSD: 2, FundingType: QuotaFundingTypeGift, Remark: "campaign"}, 1, operator)
				require.NoError(t, err)
				var voucher Redemption
				require.NoError(t, db.Where(commonKeyCol+" = ?", keys[0]).First(&voucher).Error)
				assert.Equal(t, "Free $2.00", voucher.Name)
				assert.Equal(t, 1000000, voucher.Quota)
				require.NoError(t, db.First(&user, user.Id).Error)
				beforeGift, beforePaid := user.GiftQuota, user.PaidQuota
				_, err = Redeem(keys[0], user.Id)
				require.NoError(t, err)
				require.NoError(t, db.First(&user, user.Id).Error)
				assert.Equal(t, beforeGift+1000000, user.GiftQuota)
				assert.Equal(t, beforePaid, user.PaidQuota)
				voucher.Status = common.RedemptionCodeStatusEnabled
				_, err = UpdateRedemptionWithAudit(voucher, true, operator)
				require.Error(t, err, "redeemed codes must never be re-enabled for a second credit")
				order := TopUp{UserId: user.Id, Amount: 1, Money: 9, Quota: 714285, ExchangeRate: 7,
					RevenueUSD: 1.285714, InputAmount: 10, Currency: "CNY", TradeNo: "snapshot-order",
					PaymentMethod: "test", PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusPending}
				require.NoError(t, db.Create(&order).Error)
				beforePaid = user.PaidQuota
				oldQuota := common.QuotaPerUnit
				common.QuotaPerUnit = 900000
				_, err = RechargeEpay(order.TradeNo, "test", "")
				common.QuotaPerUnit = oldQuota
				require.NoError(t, err)
				require.NoError(t, db.First(&user, user.Id).Error)
				assert.Equal(t, beforePaid+714285, user.PaidQuota, "settlement uses the order quota, not new pricing settings")
				t.Run("self_wallet_balances_after_epay", func(t *testing.T) {
					require.Positive(t, user.PaidQuota)
					require.Positive(t, user.GiftQuota)
					profile, err := GetSelfUserById(user.Id)
					require.NoError(t, err)
					assert.Equal(t, user.Quota, profile.Quota)
					assert.Equal(t, user.PaidQuota, profile.PaidQuota)
					assert.Equal(t, user.GiftQuota, profile.GiftQuota)
					assert.Equal(t, profile.Quota, profile.PaidQuota+profile.GiftQuota)
				})
				var funding UserQuotaFunding
				require.NoError(t, db.Where("source_ref_id = ? AND source_type = ?", order.Id, QuotaFundingSourceOnlineTopUp).First(&funding).Error)
				assert.InDelta(t, 1.285714, funding.RecognizedRevenueUSDTotal, 0.000001)
				beforeSummary, err := BuildFinanceRevenueSummary("month", month)
				require.NoError(t, err)
				require.NoError(t, AdjustUserQuotaWithAudit(UserQuotaAdjustmentParams{
					UserId: user.Id, FundingType: QuotaFundingTypePaid, DeltaQuota: -500000,
					SourceType: QuotaFundingSourceRefund, RevenueUSD: -0.8,
					OperatorUserId: user.Id, Remark: "partial cash refund",
				}))
				afterSummary, err := BuildFinanceRevenueSummary("month", month)
				require.NoError(t, err)
				assert.InDelta(t, beforeSummary.PaidRechargeTotal.USD-0.8, afterSummary.PaidRechargeTotal.USD, 0.000001)
				audit = FinancialAuditLog{}
				require.NoError(t, db.Last(&audit).Error)
				assert.Contains(t, audit.AfterJSON, `"revenue_usd":-0.8`)
				require.NoError(t, db.First(&user, user.Id).Error)
				assert.Equal(t, beforePaid+714285-500000, user.PaidQuota)
				for _, kind := range []string{QuotaFundingTypeGift, QuotaFundingTypePaid} {
					cny := Redemption{FundingType: kind, Remark: "currency regression"}
					require.NoError(t, cny.SetFinanceFaceValue("CNY", 70))
					require.NoError(t, cny.ValidateFinance())
					assert.Equal(t, 5000000, cny.Quota)
					usd := Redemption{FundingType: kind, Remark: "currency regression"}
					require.NoError(t, usd.SetFinanceFaceValue("USD", 10))
					require.NoError(t, usd.ValidateFinance())
					assert.Equal(t, cny.Quota, usd.Quota)
					keys, err := CreateRedemptionsWithAudit(cny, 1, operator)
					require.NoError(t, err)
					var saved Redemption
					require.NoError(t, db.Where(commonKeyCol+" = ?", keys[0]).First(&saved).Error)
					assert.Equal(t, 70.0, saved.InputAmount)
					assert.Equal(t, "CNY", saved.Currency)
					operation_setting.USDExchangeRate = 8
					quota, err := Redeem(keys[0], user.Id)
					require.NoError(t, err)
					assert.Equal(t, 5000000, quota, "redeem must use the issued card snapshot")
					operation_setting.USDExchangeRate = 7
				}
			})
		})
	}
}

// Opt-in replay copies one account from a read-only local backup into an empty
// isolated database. It never uses or modifies the production database.
func TestFinanceHappydongReplay(t *testing.T) {
	sourceDSN, targetDSN := os.Getenv("FINANCE_REPLAY_SOURCE_DSN"), os.Getenv("FINANCE_REPLAY_TARGET_DSN")
	if sourceDSN == "" || targetDSN == "" {
		t.Skip("local historical replay databases are not configured")
	}
	for _, dsn := range []string{sourceDSN, targetDSN} {
		location, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", location.Hostname())
		require.Equal(t, "55447", location.Port())
	}
	targetLocation, err := url.Parse(targetDSN)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(targetLocation.Path, "/finance_test_"))
	require.NotEqual(t, sourceDSN, targetDSN)
	sourceDB, err := gorm.Open(postgres.Open(sourceDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	targetDB, err := gorm.Open(postgres.Open(targetDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, db := range []*gorm.DB{sourceDB, targetDB} {
		connection, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, connection.Close()) })
	}
	for range 2 {
		require.NoError(t, targetDB.AutoMigrate(FinanceMonthlyModels()...))
		require.NoError(t, targetDB.AutoMigrate(&User{}, &Log{}, &Token{}, &Channel{}, &UserBalanceLedger{}, &ChannelCostLedger{}, &UserQuotaFunding{}, &FinancialAuditLog{}))
	}
	var existing int64
	require.NoError(t, targetDB.Model(&User{}).Count(&existing).Error)
	require.Zero(t, existing, "replay requires an empty dedicated database")
	var account User
	var logs []*Log
	var ledgers []ChannelCostLedger
	var balances []UserBalanceLedger
	var fundings []UserQuotaFunding
	var audits []FinancialAuditLog
	var tokens []Token
	var channels []Channel
	var firstActivity int64
	err = sourceDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Select("id,username,quota,paid_quota,gift_quota,used_quota,created_at").Where("username = ?", "happydong").First(&account).Error; err != nil {
			return err
		}
		if err := tx.Model(&Log{}).Where("user_id = ?", account.Id).Select("MIN(created_at)").Scan(&firstActivity).Error; err != nil {
			return err
		}
		if err := tx.Select("id,user_id,type,created_at,quota,request_id,token_name,token_id,channel_id,model_name,other").Where("user_id = ? AND type IN ?", account.Id, []int{LogTypeConsume, LogTypeRefund}).Find(&logs).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", account.Id).Find(&ledgers).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", account.Id).Find(&balances).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", account.Id).Find(&fundings).Error; err != nil {
			return err
		}
		if err := tx.Where("target_user_id = ? AND module = ? AND target_type = ?", account.Id, FinancialAuditModuleUserQuota, FinancialAuditTargetUserWallet).Find(&audits).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Select("id,user_id,name").Where("user_id = ?", account.Id).Find(&tokens).Error; err != nil {
			return err
		}
		return tx.Select("id,name").Where("id IN (?)", tx.Model(&Log{}).Where("user_id = ?", account.Id).Select("channel_id")).Find(&channels).Error
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	require.NoError(t, err)
	account.AffCode = "isolated-historical-replay"
	for i := range tokens {
		tokens[i].Key = fmt.Sprintf("disabled-replay-token-%d", tokens[i].Id)
		tokens[i].Status = common.TokenStatusDisabled
	}
	for _, log := range logs {
		other, err := common.Marshal(map[string]string{"billing_source": legacyBillingSource(parseStatementOther(log.Other))})
		require.NoError(t, err)
		log.Other = string(other)
	}
	require.NoError(t, targetDB.Create(&account).Error)
	for _, rows := range []any{logs, ledgers, balances, fundings, audits, tokens, channels} {
		require.NoError(t, targetDB.CreateInBatches(rows, 200).Error)
	}
	oldDB, oldLogDB, oldQuota := DB, LOG_DB, common.QuotaPerUnit
	oldMain, oldLog := common.MainDatabaseType(), common.LogDatabaseType()
	DB, LOG_DB, common.QuotaPerUnit = targetDB, targetDB, 500000
	common.SetDatabaseTypes(common.DatabaseType("postgres"), common.DatabaseType("postgres"))
	initCol()
	t.Cleanup(func() {
		DB, LOG_DB, common.QuotaPerUnit = oldDB, oldLogDB, oldQuota
		common.SetDatabaseTypes(oldMain, oldLog)
		initCol()
	})
	out := os.Getenv("FINANCE_REPLAY_OUTPUT")
	require.NotEmpty(t, out)
	require.NoError(t, os.MkdirAll(out, 0755))
	summaryRows := [][]string{{"月份", "上期余额(CNY)", "充值(CNY)", "赠送(CNY)", "调账(CNY)", "消费退回(CNY)", "总消费(CNY)", "本期余额(CNY)", "对账差额(CNY)", "原始日志消费Quota", "账单消费Quota", "消费差异Quota", "补入旧日志笔数", "说明"}}
	now := time.Date(2026, 10, 18, 20, 0, 0, 0, financeBillingLocation)
	t.Logf("first activity: %s; wallet rows=%d; cost rows=%d; logs=%d; audit checkpoints=%d", time.Unix(firstActivity, 0).In(financeBillingLocation).Format(time.DateTime), len(balances), len(ledgers), len(logs), len(audits))
	for m := time.May; m <= time.September; m++ {
		start := time.Date(2026, m, 1, 0, 0, 0, 0, financeBillingLocation)
		month := newFinanceBillingMonth(start, 7, 1000, 500000, now)
		require.NoError(t, DB.Create(&month).Error)
		require.NoError(t, closeFinanceBillingMonth(month, now))
		var count int64
		require.NoError(t, DB.Model(&FinanceMonthlyBill{}).Where("bill_month = ? AND user_id = ?", month.BillMonth, account.Id).Count(&count).Error)
		if count == 0 {
			assert.LessOrEqual(t, month.EndsAt, firstActivity)
			summaryRows = append(summaryRows, []string{month.BillMonth, "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0", "0", "0", "0", "无开户及资金记录，不生成虚构固定账单"})
			continue
		}
		bill, err := GetFixedFinanceBill(month.BillMonth, account.Id)
		require.NoError(t, err)
		require.NotNil(t, bill.WalletSummary)
		var items []FinanceMonthlyBillItem
		require.NoError(t, DB.Where("bill_id = ?", bill.Id).Order("id").Find(&items).Error)
		var rawQuota int64
		require.NoError(t, DB.Model(&Log{}).Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?", account.Id, LogTypeConsume, month.StartsAt, month.EndsAt).Select("COALESCE(SUM(quota),0)").Scan(&rawQuota).Error)
		var billQuota int64
		fallback := 0
		for _, item := range items {
			if item.EntryType == CustomerMonthlyStatementEntryTypeConsume && item.BillingSource != "subscription" {
				billQuota -= int64(item.QuotaRaw)
				if item.SourceTable == CustomerMonthlyStatementSourceTableLogs {
					fallback++
				}
			}
		}
		assert.Equal(t, rawQuota, billQuota, "original log totals independently verify the merged statement")
		var creditedQuota int64
		require.NoError(t, DB.Model(&UserBalanceLedger{}).Where("user_id = ? AND amount_quota > 0 AND occurred_at >= ? AND occurred_at < ?", account.Id, month.StartsAt, month.EndsAt).Select("COALESCE(SUM(amount_quota),0)").Scan(&creditedQuota).Error)
		s := bill.WalletSummary
		assert.InDelta(t, float64(creditedQuota)*7/500000, s.FundsIn.CNY, 0.00000001)
		assert.InDelta(t, float64(rawQuota)*7/500000, s.Consumption.CNY, 0.00000001)
		assert.InDelta(t, s.Closing.CNY, s.Opening.CNY+s.ExchangeChangeCNY+s.FundsIn.CNY-s.FundsOut.CNY+s.Reconciliation.CNY, 0.0000001)
		content, err := ExportFixedFinanceBill(month.BillMonth, account.Id)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(out, "happydong-"+month.BillMonth+".csv"), content, 0600))
		var saved FinanceMonthlyBillFile
		require.NoError(t, DB.First(&saved, "bill_id = ?", bill.Id).Error)
		assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(content)), saved.SHA256)
		require.NoError(t, closeFinanceBillingMonth(month, now.Add(time.Minute)))
		again, err := ExportFixedFinanceBill(month.BillMonth, account.Id)
		require.NoError(t, err)
		assert.Equal(t, content, again, "reruns cannot replace a frozen statement")
		summaryRows = append(summaryRows, []string{month.BillMonth, fmt.Sprintf("%.2f", s.Opening.CNY), fmt.Sprintf("%.2f", s.Topup.CNY), fmt.Sprintf("%.2f", s.Gift.CNY), fmt.Sprintf("%.2f", s.Adjustment.CNY), fmt.Sprintf("%.2f", s.Refund.CNY), fmt.Sprintf("%.2f", s.Consumption.CNY), fmt.Sprintf("%.2f", s.Closing.CNY), fmt.Sprintf("%.2f", s.Reconciliation.CNY), fmt.Sprint(rawQuota), fmt.Sprint(billQuota), fmt.Sprint(billQuota - rawQuota), fmt.Sprint(fallback), "回放汇率7，历史月度汇率尚待确认；差额保留不改余额"})
		t.Logf("%s recharge=%.2f gift=%.2f consumption=%.2f opening=%.2f closing=%.2f reconciliation=%.8f quota_difference=%d fallback=%d", month.BillMonth, s.Topup.CNY, s.Gift.CNY, s.Consumption.CNY, s.Opening.CNY, s.Closing.CNY, s.Reconciliation.CNY, billQuota-rawQuota, fallback)
	}
	file, err := os.Create(filepath.Join(out, "happydong-reconciliation.csv"))
	require.NoError(t, err)
	_, err = file.WriteString("\xEF\xBB\xBF")
	require.NoError(t, err)
	writer := csvreader.NewWriter(file)
	writer.UseCRLF = true
	require.NoError(t, writer.WriteAll(summaryRows))
	require.NoError(t, file.Close())
	var unchanged User
	require.NoError(t, sourceDB.Unscoped().Select("id,quota,paid_quota,gift_quota,used_quota").First(&unchanged, account.Id).Error)
	assert.Equal(t, account.Quota, unchanged.Quota)
	assert.Equal(t, account.PaidQuota, unchanged.PaidQuota)
	assert.Equal(t, account.GiftQuota, unchanged.GiftQuota)
	assert.Equal(t, account.UsedQuota, unchanged.UsedQuota)
}

func testFixedMonthlyBilling(t *testing.T, db *gorm.DB, userID int) {
	oldRate := operation_setting.USDExchangeRate
	oldCOS := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = oldRate
		operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate = oldCOS
	})
	operation_setting.USDExchangeRate = 7
	operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate = 1000
	october := time.Date(2024, 10, 1, 0, 0, 0, 0, financeBillingLocation)
	closeTime := october.AddDate(0, 1, 0).Add(30 * time.Minute)
	require.NoError(t, EnsureFinanceBillingMonths(october.AddDate(0, 0, 15)))
	require.ErrorIs(t, SetFinanceNextMonthRate("2024-10", 8, userID, october), ErrFinanceRateLocked)
	require.Error(t, SetFinanceNextMonthRate("2024-11", math.NaN(), userID, october))
	require.Error(t, SetFinanceNextMonthRate("2024-11", -1, userID, october))
	require.NoError(t, SetFinanceNextMonthRate("2024-11", 7.3, userID, october))
	var beforeLegacy int64
	require.NoError(t, db.Model(&CustomerMonthlyStatement{}).Count(&beforeLegacy).Error)
	for _, ledger := range []ChannelCostLedger{
		{UserId: userID, RequestId: "month-october", EntryType: "consume", ActualQuota: 5000000, InternalEquivalentUSD: 10,
			OccurredAt: closeTime.Add(-31 * time.Minute).Unix(), CreatedAt: closeTime.Add(-31 * time.Minute).Unix()},
		{UserId: userID, RequestId: "month-november", EntryType: "consume", ActualQuota: 5000000, InternalEquivalentUSD: 10,
			OccurredAt: closeTime.Add(-30 * time.Minute).Unix(), CreatedAt: closeTime.Add(-30 * time.Minute).Unix()},
		{UserId: userID, RequestId: "month-late", EntryType: "refund", ActualQuota: 500000, InternalEquivalentUSD: 1,
			OccurredAt: closeTime.Add(-31 * time.Minute).Unix(), CreatedAt: closeTime.Add(5 * time.Second).Unix()},
	} {
		require.NoError(t, db.Create(&ledger).Error)
	}
	// The cutoff uses Asia/Shanghai even when the scheduler receives UTC.
	require.NoError(t, RunFinanceMonthlyBilling(closeTime.Add(-time.Second).UTC()))
	_, err := GetFixedFinanceBill("2024-10", userID)
	require.ErrorIs(t, err, ErrFinanceMonthNotClosed)
	operation_setting.USDExchangeRate = 99
	operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate = 99
	require.NoError(t, RunFinanceMonthlyBilling(closeTime.UTC()))
	bill, err := GetFixedFinanceBill("2024-10", userID)
	require.NoError(t, err)
	assert.Equal(t, 10.0, bill.ConsumeUSD)
	assert.Equal(t, 70.0, bill.ConsumeCNY)
	assert.Equal(t, 10000.0, bill.ConsumeCOS)
	assert.Equal(t, 7.0, bill.CNYRate)
	export, err := ExportFixedFinanceBill("2024-10", userID)
	require.NoError(t, err)
	assert.NotContains(t, string(export), "COS")
	assert.NotContains(t, string(export), "Quota")
	assert.Contains(t, string(export), "70.00")
	var file FinanceMonthlyBillFile
	require.NoError(t, db.Where("bill_id = ?", bill.Id).First(&file).Error)
	assert.Equal(t, export, file.Content, "download returns the stored file bytes")
	current, err := ExportCurrentFinanceBill(userID, closeTime.Add(time.Minute))
	require.NoError(t, err)
	assert.Contains(t, string(current), "本月明细，未出账")
	assert.Contains(t, string(current), "余额变动(CNY)")
	assert.Contains(t, string(current), "73.00")
	assert.NotContains(t, string(current), "COS")
	assert.NotContains(t, string(current), "Quota")
	require.NoError(t, db.Model(&User{}).Where("id = ?", userID).Update("username", "renamed-finance-test").Error)
	require.NoError(t, db.Model(&ChannelCostLedger{}).Where("request_id = ?", "month-october").Update("internal_equivalent_usd", 123).Error)
	require.NoError(t, RunFinanceMonthlyBilling(closeTime.Add(time.Hour)))
	after, err := ExportFixedFinanceBill("2024-10", userID)
	require.NoError(t, err)
	assert.Equal(t, export, after, "retries, renames and new global rates cannot rewrite a fixed bill")
	page, err := ListFixedFinanceBills("2024-10", "", userID+10000, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Zero(t, page.Total)
	var afterLegacy int64
	require.NoError(t, db.Model(&CustomerMonthlyStatement{}).Count(&afterLegacy).Error)
	assert.Equal(t, beforeLegacy, afterLegacy, "legacy statements are preserved")
	require.ErrorIs(t, SetFinanceNextMonthRate("2024-11", 8, userID, closeTime), ErrFinanceRateLocked)

	// Fail a row insertion once: the whole customer's bill must roll back and
	// the month stays private until the automatic retry succeeds.
	failInsert := true
	const callback = "finance:test_monthly_failure"
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if failInsert && tx.Statement.Table == "finance_monthly_bill_items" {
			tx.AddError(fmt.Errorf("injected local statement failure"))
		}
	}))
	t.Cleanup(func() { db.Callback().Create().Remove(callback) })
	decemberClose := closeTime.AddDate(0, 1, 0)
	require.Error(t, RunFinanceMonthlyBilling(decemberClose))
	_, err = GetFixedFinanceBill("2024-11", userID)
	require.ErrorIs(t, err, ErrFinanceMonthNotClosed)
	failInsert = false
	require.NoError(t, RunFinanceMonthlyBilling(decemberClose.Add(time.Minute)))
	november, err := GetFixedFinanceBill("2024-11", userID)
	require.NoError(t, err)
	assert.Equal(t, 9.0, november.ConsumeUSD)
	assert.Equal(t, 65.7, november.ConsumeCNY)
	assert.Equal(t, 2, november.ItemCount, "late refund carries forward once")
	require.NoError(t, RunFinanceMonthlyBilling(decemberClose.AddDate(0, 2, 0)))
	var inherited FinanceBillingMonth
	require.NoError(t, db.Where("bill_month = ?", "2025-02").First(&inherited).Error)
	assert.Equal(t, 7.3, inherited.CNYRate, "missed months inherit the most recent scheduled rate")
}
