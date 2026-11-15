package model

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPic365MigrationAndGrants(t *testing.T) {
	for _, engine := range []string{"sqlite", "postgres", "mysql"} {
		t.Run(engine, func(t *testing.T) {
			var dialect gorm.Dialector = sqlite.Open(":memory:")
			if engine == "postgres" {
				dsn := os.Getenv("FINANCE_TEST_PG_DSN")
				if dsn == "" {
					t.Skip("isolated local PostgreSQL is not configured")
				}
				require.True(t, strings.Contains(dsn, "127.0.0.1") && strings.Contains(dsn, "/finance_test_pic365"))
				dialect = postgres.Open(dsn)
			}
			if engine == "mysql" {
				dsn := os.Getenv("FINANCE_TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("isolated local MySQL is not configured")
				}
				require.True(t, strings.Contains(dsn, "127.0.0.1") && strings.Contains(dsn, "/finance_test_pic365"))
				dialect = mysql.Open(dsn)
			}
			db, err := gorm.Open(dialect, &gorm.Config{})
			require.NoError(t, err)
			connection, err := db.DB()
			require.NoError(t, err)
			connection.SetMaxOpenConns(1)
			oldDB, oldType := DB, common.MainDatabaseType()
			DB = db
			common.SetDatabaseTypes(common.DatabaseType(engine), common.LogDatabaseType())
			initCol()
			t.Cleanup(func() {
				DB = oldDB
				common.SetDatabaseTypes(oldType, common.LogDatabaseType())
				initCol()
				require.NoError(t, connection.Close())
			})
			models := []any{&User{}, &Token{}, &UserSession{}, &AuthFlow{}, &Pic365Grant{}}
			require.NoError(t, db.Migrator().DropTable(&Pic365Grant{}, &AuthFlow{}, &UserSession{}, &Token{}, &User{}))
			for range 2 {
				require.NoError(t, db.AutoMigrate(models...))
			}
			var version string
			query := "SELECT version()"
			if engine == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database: %s", version)

			// Upgrade the released account/key schema and keep preexisting balances.
			require.NoError(t, db.Migrator().DropColumn(&User{}, "Pic365Subject"))
			require.NoError(t, db.Migrator().DropColumn(&Token{}, "DelegatedClient"))
			require.NoError(t, db.Migrator().DropTable(&Pic365Grant{}))
			legacy := User{Username: "old-account", AuthVersion: 1, Status: common.UserStatusEnabled, Quota: 500000, PaidQuota: 500000, AffCode: "local-old-aff-code"}
			require.NoError(t, db.Omit("Pic365Subject").Create(&legacy).Error)
			legacyKey := Token{UserId: legacy.Id, Key: "old-unchanged-key", Name: "old-key", Status: common.TokenStatusEnabled}
			require.NoError(t, db.Omit("DelegatedClient").Create(&legacyKey).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(models...))
			}
			var saved User
			require.NoError(t, db.First(&saved, legacy.Id).Error)
			assert.Equal(t, 500000, saved.PaidQuota)
			assert.Empty(t, saved.Pic365Subject)
			var original Token
			require.NoError(t, db.First(&original, legacyKey.Id).Error)
			assert.Equal(t, "old-unchanged-key", original.Key)
			assert.Empty(t, original.DelegatedClient)

			session := UserSession{SID: uuid.NewString(), UserID: saved.Id, Version: 1, UserAuthVersion: 1, Status: UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}
			require.NoError(t, db.Create(&session).Error)
			identity := AuthSessionIdentity{UserID: saved.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1}
			var key *Token
			var subject string
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				var err error
				key, subject, err = CreatePic365GrantWithTx(tx, identity, "gpt-image-2", time.Now().Add(24*time.Hour).Unix())
				return err
			}))
			require.NoError(t, ValidatePic365Grant(key))
			assert.Equal(t, session.ExpiresAt, key.ExpiredTime)
			assert.NotEmpty(t, subject)
			assert.Equal(t, "default", key.Group)
			require.NoError(t, db.First(&saved, saved.Id).Error)
			assert.Equal(t, 500000, saved.PaidQuota, "issuing credentials does not debit the wallet")
			t.Run("Redis retains delegation and upgrades old hashes without overwriting quota", func(t *testing.T) {
				server := useUserCacheMiniRedis(t)
				initialized, err := cacheInitToken(*key)
				require.NoError(t, err)
				require.Equal(t, 1, initialized)
				cached, err := GetTokenByKey(key.Key, false)
				require.NoError(t, err)
				assert.Equal(t, Pic365Client, cached.DelegatedClient)
				require.NoError(t, ValidatePic365Grant(cached))
				server.HDel(getTokenCacheKey(key.Key), "DelegatedClient")
				_, err = cacheGetTokenByKey(key.Key)
				require.Error(t, err, "missing delegation metadata must not be treated as an ordinary key")
				server.HSet(getTokenCacheKey(key.Key), "RemainQuota", "37")
				server.HSet(getTokenCacheKey(key.Key), "UsedQuota", "18")
				_, err = GetTokenByKey(key.Key, false)
				require.NoError(t, err)
				cached, err = cacheGetTokenByKey(key.Key)
				require.NoError(t, err)
				assert.Equal(t, Pic365Client, cached.DelegatedClient)
				assert.Equal(t, 37, cached.RemainQuota)
				assert.Equal(t, 18, cached.UsedQuota)
				require.NoError(t, ValidatePic365Grant(cached))
			})
			t.Run("renewal preserves an in-flight token snapshot", func(t *testing.T) {
				snapshot := *key
				snapshot.ExpiredTime = time.Now().Add(45 * time.Second).Unix()
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("expired_time", snapshot.ExpiredTime).Error)
				require.NoError(t, db.Model(&Pic365Grant{}).Where("token_id = ?", key.Id).Update("expires_at", snapshot.ExpiredTime).Error)
				binding, err := GetPic365GrantBinding(&snapshot)
				require.NoError(t, err)
				next, nextBinding, err := RenewPic365Grant(key.Key, binding)
				require.NoError(t, err)
				assert.Greater(t, next.ExpiredTime, snapshot.ExpiredTime)
				assert.Equal(t, binding, nextBinding)
				require.NoError(t, ValidatePic365Grant(&snapshot))
				assert.True(t, Pic365GrantBindingMatches(&snapshot, binding))
				key = next
			})
			t.Run("renewal after sleep retains key binding and accounting", func(t *testing.T) {
				beforeBinding, err := GetPic365GrantBinding(key)
				require.NoError(t, err)
				usedQuota := key.UsedQuota
				expiredAt := time.Now().Add(-time.Minute).Unix()
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Updates(map[string]any{"expired_time": expiredAt, "status": common.TokenStatusExpired}).Error)
				require.NoError(t, db.Model(&Pic365Grant{}).Where("token_id = ?", key.Id).Update("expires_at", expiredAt).Error)
				next, nextBinding, err := RenewPic365Grant(key.Key, beforeBinding)
				require.NoError(t, err)
				assert.Equal(t, key.Key, next.Key)
				assert.Equal(t, key.Id, next.Id)
				assert.Equal(t, usedQuota, next.UsedQuota)
				assert.Equal(t, beforeBinding, nextBinding, "normal renewal must not invalidate in-flight requests")
				assert.InDelta(t, time.Now().Add(5*time.Minute).Unix(), next.ExpiredTime, 2)
				require.NoError(t, ValidatePic365Grant(next))
				key = next
				var grant Pic365Grant
				require.NoError(t, db.First(&grant, "token_id = ?", key.Id).Error)
				legacyBinding := pic365GrantBinding(key, &grant, true)
				assert.True(t, Pic365GrantBindingMatches(key, legacyBinding), "already signed-in clients remain compatible")
				next, upgradedBinding, err := RenewPic365Grant(key.Key, legacyBinding)
				require.NoError(t, err)
				assert.Equal(t, nextBinding, upgradedBinding)
				key = next
				require.NoError(t, db.First(&saved, saved.Id).Error)
				assert.Equal(t, 500000, saved.PaidQuota)
				for _, mutation := range []struct {
					column string
					value  any
				}{{"revoked_at", time.Now().Unix()}, {"expires_at", expiredAt}, {"version", int64(2)}} {
					require.NoError(t, db.Model(&UserSession{}).Where("sid = ?", session.SID).Update(mutation.column, mutation.value).Error)
					_, _, err = RenewPic365Grant(key.Key, nextBinding)
					assert.ErrorIs(t, err, ErrPic365GrantInvalid)
					require.NoError(t, db.Model(&UserSession{}).Where("sid = ?", session.SID).Updates(map[string]any{"revoked_at": 0, "expires_at": session.ExpiresAt, "version": 1}).Error)
				}
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("auth_version", 2).Error)
				_, _, err = RenewPic365Grant(key.Key, nextBinding)
				assert.ErrorIs(t, err, ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("auth_version", 1).Error)
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("status", common.TokenStatusDisabled).Error)
				_, _, err = RenewPic365Grant(key.Key, nextBinding)
				assert.ErrorIs(t, err, ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("status", common.TokenStatusEnabled).Error)
				_, _, err = RenewPic365Grant(key.Key, strings.Repeat("x", 43))
				assert.ErrorIs(t, err, ErrPic365GrantInvalid)
			})
			t.Run("simultaneous launches cannot allocate duplicate keys", func(t *testing.T) {
				if engine != "sqlite" {
					connection.SetMaxOpenConns(5)
					defer connection.SetMaxOpenConns(1)
				}
				type result struct {
					token *Token
					err   error
				}
				results := make(chan result, 5)
				for range 5 {
					go func() {
						var next *Token
						err := db.Transaction(func(tx *gorm.DB) error {
							var err error
							next, _, err = CreatePic365GrantWithTx(tx, identity, "gpt-image-2", session.ExpiresAt)
							return err
						})
						results <- result{token: next, err: err}
					}()
				}
				for range 5 {
					next := <-results
					require.NoError(t, next.err)
					assert.Equal(t, key.Id, next.token.Id)
					assert.Equal(t, key.Key, next.token.Key)
					key = next.token
				}
			})

			t.Run("launches reuse one default key and keep accounting", func(t *testing.T) {
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("used_quota", 1234).Error)
				beforeBinding, err := GetPic365GrantBinding(key)
				require.NoError(t, err)
				for range 3 {
					require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
						next, nextSubject, err := CreatePic365GrantWithTx(tx, identity, "gpt-image-2,gemini-3.1-flash-image-preview,grok-imagine-image-2.0", time.Now().Add(24*time.Hour).Unix())
						if err != nil {
							return err
						}
						assert.Equal(t, key.Id, next.Id)
						assert.Equal(t, key.Key, next.Key)
						assert.Equal(t, subject, nextSubject)
						assert.Equal(t, 1234, next.UsedQuota)
						key = next
						return nil
					}))
				}
				var count int64
				require.NoError(t, db.Model(&Token{}).Where("user_id = ? AND delegated_client = ?", saved.Id, Pic365Client).Count(&count).Error)
				assert.EqualValues(t, 1, count)
				binding, err := GetPic365GrantBinding(key)
				require.NoError(t, err)
				assert.NotEqual(t, beforeBinding, binding, "changed model authorization invalidates the old binding without changing the key")
				require.NoError(t, db.First(&original, original.Id).Error)
				assert.Equal(t, "old-unchanged-key", original.Key)
			})
			t.Run("legacy bridge duplicates consolidate without touching ordinary keys", func(t *testing.T) {
				duplicate := Token{UserId: saved.Id, Key: "legacy-pic365-duplicate", Name: "Pic365", Status: common.TokenStatusEnabled, DelegatedClient: Pic365Client}
				require.NoError(t, db.Create(&duplicate).Error)
				grant := Pic365Grant{TokenID: duplicate.Id, UserID: saved.Id, Subject: subject, SessionID: session.SID,
					AuthVersion: 1, SessionVersion: 1, Models: "gpt-image-2", ExpiresAt: session.ExpiresAt}
				require.NoError(t, db.Create(&grant).Error)
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					var err error
					key, _, err = CreatePic365GrantWithTx(tx, identity, "gpt-image-2", session.ExpiresAt)
					return err
				}))
				assert.Equal(t, duplicate.Id, key.Id)
				var count int64
				require.NoError(t, db.Model(&Token{}).Where("user_id = ? AND delegated_client = ?", saved.Id, Pic365Client).Count(&count).Error)
				assert.EqualValues(t, 1, count)
				require.NoError(t, db.First(&original, original.Id).Error)
				assert.Empty(t, original.DelegatedClient)
			})

			t.Run("cannot widen model scope", func(t *testing.T) {
				changed := *key
				changed.ModelLimitsEnabled = false
				assert.ErrorIs(t, ValidatePic365Grant(&changed), ErrPic365GrantInvalid)
				changed = *key
				changed.Group = "cheaper-group"
				assert.ErrorIs(t, ValidatePic365Grant(&changed), ErrPic365GrantInvalid)
			})
			t.Run("disabled source and logout revoke the grant", func(t *testing.T) {
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("status", common.UserStatusDisabled).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("status", common.UserStatusEnabled).Error)
				require.NoError(t, db.Model(&UserSession{}).Where("sid = ?", session.SID).Update("revoked_at", time.Now().Unix()).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&UserSession{}).Where("sid = ?", session.SID).Update("revoked_at", 0).Error)
			})
			t.Run("password version invalidates and expiration rejects", func(t *testing.T) {
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("auth_version", 2).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&User{}).Where("id = ?", saved.Id).Update("auth_version", 1).Error)
				require.NoError(t, db.Model(&Pic365Grant{}).Where("token_id = ?", key.Id).Update("expires_at", time.Now().Add(-time.Second).Unix()).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				require.NoError(t, db.Model(&Pic365Grant{}).Where("token_id = ?", key.Id).Update("expires_at", key.ExpiredTime).Error)
			})
			t.Run("disabled stored key rejects cached credentials", func(t *testing.T) {
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("status", common.TokenStatusDisabled).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				assert.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
					_, _, err := CreatePic365GrantWithTx(tx, identity, "gpt-image-2", session.ExpiresAt)
					return err
				}), ErrPic365GrantInvalid, "a launch must not undo manual revocation")
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Update("status", common.TokenStatusEnabled).Error)
				require.NoError(t, ValidatePic365Grant(key))
			})
			t.Run("expired authorization renews the same key after a valid dashboard launch", func(t *testing.T) {
				require.NoError(t, db.Model(&Token{}).Where("id = ?", key.Id).Updates(map[string]any{"status": common.TokenStatusExpired, "expired_time": time.Now().Add(-time.Hour).Unix()}).Error)
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					next, _, err := CreatePic365GrantWithTx(tx, identity, key.ModelLimits, session.ExpiresAt)
					if err == nil {
						assert.Equal(t, key.Key, next.Key)
						assert.Equal(t, key.Id, next.Id)
						key = next
					}
					return err
				}))
				require.NoError(t, ValidatePic365Grant(key))
			})
			t.Run("a new dashboard session changes authorization not the dedicated key", func(t *testing.T) {
				oldBinding, err := GetPic365GrantBinding(key)
				require.NoError(t, err)
				nextSession := session
				nextSession.SID = uuid.NewString()
				require.NoError(t, db.Create(&nextSession).Error)
				nextIdentity := identity
				nextIdentity.SessionID = nextSession.SID
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					next, _, err := CreatePic365GrantWithTx(tx, nextIdentity, key.ModelLimits, session.ExpiresAt)
					if err == nil {
						assert.Equal(t, key.Key, next.Key)
						assert.Equal(t, key.Id, next.Id)
						key = next
					}
					return err
				}))
				nextBinding, err := GetPic365GrantBinding(key)
				require.NoError(t, err)
				assert.NotEqual(t, oldBinding, nextBinding)
				_, _, err = RenewPic365Grant(key.Key, oldBinding)
				assert.ErrorIs(t, err, ErrPic365GrantInvalid, "renewal cannot revive a different source session")
			})
			t.Run("numeric ID reuse does not inherit the old subject", func(t *testing.T) {
				require.NoError(t, db.Unscoped().Delete(&User{}, saved.Id).Error)
				replacement := User{Id: saved.Id, Username: "replacement", AuthVersion: 1, Status: common.UserStatusEnabled, AffCode: "local-replacement-aff-code"}
				require.NoError(t, db.Create(&replacement).Error)
				assert.ErrorIs(t, ValidatePic365Grant(key), ErrPic365GrantInvalid)
				assert.Empty(t, replacement.Pic365Subject)
			})
		})
	}
}
