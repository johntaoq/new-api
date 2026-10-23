package controller

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSummaryAndDetailDatabaseMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		kind, env string
		typ       common.DatabaseType
	}{
		{"sqlite", "", common.DatabaseTypeSQLite},
		{"mysql", "AUDIT_MYSQL_DSN", common.DatabaseTypeMySQL},
		{"postgres", "AUDIT_POSTGRES_DSN", common.DatabaseTypePostgreSQL},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if tc.env != "" && dsn == "" {
				t.Skip(tc.env + " is not configured")
			}
			db, _ := newAuditTestDatabase(t, tc.kind, dsn)
			useLogExportDatabases(t, db, db, tc.typ, tc.typ)
			require.NoError(t, db.AutoMigrate(&model.Log{}, &model.Channel{}))
			now := time.Now().Unix()
			other := `{"model_ratio":1.5,"cache_tokens":10,"billing_source":"wallet","stream_status":{"status":"error","end_reason":"upstream_error","errors":["full-stream-error"]},"po":["` + strings.Repeat("audit-payload", 6000) + `"],"admin_info":{"use_channel":["9"],"reject_reason":"admin-only"},"root_info":{"private":"root-only"},"audit_info":{"path":"/private"}}`
			require.NoError(t, db.Create(&model.Channel{Id: 9, Name: "primary"}).Error)
			entry := model.Log{Id: 41, UserId: 1, CreatedAt: now, Type: model.LogTypeConsume, ChannelId: 9, Quota: 1234, PromptTokens: 200, CompletionTokens: 30, RequestId: "detail-test", Content: strings.Repeat("long content ", 200), Other: other}
			require.NoError(t, db.Create(&entry).Error)
			legacy := model.Log{Id: 42, UserId: 1, CreatedAt: now, Type: model.LogTypeManage, Content: "legacy log", Other: "{}"}
			require.NoError(t, db.Create(&legacy).Error)

			for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
				t.Run(fmt.Sprint(role), func(t *testing.T) {
					list, detail := GetUserLogs, GetUserLogDetail
					if role >= common.RoleAdminUser {
						list, detail = GetAllLogs, GetAllLogDetail
					}
					response := callLogHandler(t, list, 1, role, "summary=true&request_id=detail-test")[0]
					require.Equal(t, true, response["success"])
					items := response["data"].(map[string]any)["items"].([]any)
					require.Len(t, items, 1)
					row := items[0].(map[string]any)
					assert.Equal(t, true, row["summary"])
					assert.EqualValues(t, 41, row["detail_id"])
					assert.EqualValues(t, entry.Quota, row["quota"])
					assert.EqualValues(t, entry.PromptTokens, row["prompt_tokens"])
					assert.Less(t, len(row["content"].(string)), 1024)
					assert.NotContains(t, row["other"], "audit-payload")
					assert.NotContains(t, row["other"], "full-stream-error")
					assert.NotContains(t, row["other"], "root-only")
					assert.Contains(t, row["other"], `"cache_tokens":10`)
					assert.Contains(t, row["other"], `"model_ratio":1.5`)
					assert.Contains(t, row["other"], `"end_reason":"upstream_error"`)
					full := callLogHandler(t, detail, 1, role, "id=41&request_id=detail-test")[0]
					require.Equal(t, true, full["success"])
					data := full["data"].(map[string]any)
					assert.Equal(t, entry.Content, data["content"])
					assert.Contains(t, data["other"], "audit-payload")
					assert.Contains(t, data["other"], "full-stream-error")
					assert.NotContains(t, data, "summary")
					if role == common.RoleCommonUser {
						assert.EqualValues(t, 1, row["id"], "display position must not become the detail locator")
						assert.NotContains(t, data["other"], "admin-only")
						assert.Empty(t, data["channel_name"])
					} else {
						assert.Contains(t, data["other"], "admin-only")
						assert.Equal(t, "primary", data["channel_name"])
					}
					assert.Equal(t, role == common.RoleRootUser, strings.Contains(data["other"].(string), "root-only"))
					fullList := callLogHandler(t, list, 1, role, "request_id=detail-test")[0]
					fullRow := fullList["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
					assert.Equal(t, data["other"], fullRow["other"], "legacy list API remains compatible")
				})
			}
			for _, query := range []string{"", "id=-1", "id=oops", "id=999", "id=41&request_id=wrong", "id=41&username=other"} {
				assert.Equal(t, false, callLogHandler(t, GetUserLogDetail, 2, common.RoleCommonUser, query)[0]["success"], "a caller cannot read another user's log: %s", query)
			}
			assert.Equal(t, true, callLogHandler(t, GetUserLogDetail, 1, common.RoleCommonUser, "id=42")[0]["success"], "legacy rows without request ids remain accessible")
			assert.Equal(t, true, callLogHandler(t, GetUserLogDetail, 1, common.RoleCommonUser, fmt.Sprintf("request_id=detail-test&created_at=%d&type=2", now))[0]["success"])
			assert.Equal(t, false, callLogHandler(t, GetUserLogDetail, 1, common.RoleCommonUser, fmt.Sprintf("request_id=detail-test&created_at=%d&type=5", now))[0]["success"], "request ids can be shared by different log events")
			assert.Equal(t, false, callLogHandler(t, GetUserLogDetail, 1, common.RoleCommonUser, "request_id=detail-test&created_at=oops")[0]["success"])
			assert.NotContains(t, callLogHandler(t, GetUserLogDetail, 1, common.RoleRootUser, "id=41")[0]["data"].(map[string]any)["other"], "admin_info", "self detail always uses owner visibility, even for root")
			var stored model.Log
			require.NoError(t, db.First(&stored, 41).Error)
			assert.Equal(t, other, stored.Other, "summary projection never rewrites stored audit data")
		})
	}
}
