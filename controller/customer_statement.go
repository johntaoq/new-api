package controller

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type AdminBillingPermissions struct {
	CanAdjustGiftQuota       bool `json:"can_adjust_gift_quota"`
	CanAdjustPaidQuota       bool `json:"can_adjust_paid_quota"`
	CanEditFinancialSettings bool `json:"can_edit_financial_settings"`
}

func GenerateSelfMonthlyStatement(c *gin.Context) {
	userId := c.GetInt("id")
	billMonth := c.Query("bill_month")
	statement, err := model.GenerateCustomerMonthlyStatement(userId, billMonth, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	bundle, err := model.BuildCustomerMonthlyStatementBundle(statement, userId, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func GetSelfMonthlyStatement(c *gin.Context) {
	userId := c.GetInt("id")
	billMonth := c.Query("bill_month")
	statement, err := model.GenerateCustomerMonthlyStatement(userId, billMonth, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	bundle, err := model.BuildCustomerMonthlyStatementBundle(statement, userId, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func ListSelfMonthlyStatements(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	statements, total, err := model.ListCustomerMonthlyStatements(userId, c.Query("bill_month"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(statements)
	common.ApiSuccess(c, pageInfo)
}

func ExportSelfMonthlyStatementCSV(c *gin.Context) {
	userId := c.GetInt("id")
	statement, err := model.GenerateCustomerMonthlyStatement(userId, c.Query("bill_month"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeMonthlyStatementCSV(c, statement, userId)
}

func GenerateAdminMonthlyStatement(c *gin.Context) {
	userId, err := parseStatementUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := model.GenerateCustomerMonthlyStatement(userId, c.Query("bill_month"), true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	bundle, err := model.BuildCustomerMonthlyStatementBundle(statement, 0, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func GetAdminMonthlyStatement(c *gin.Context) {
	userId, err := parseStatementUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := model.GenerateCustomerMonthlyStatement(userId, c.Query("bill_month"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	bundle, err := model.BuildCustomerMonthlyStatementBundle(statement, 0, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func ListAdminMonthlyStatements(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := 0
	if strings.TrimSpace(c.Query("user_id")) != "" {
		parsedUserId, err := parseStatementUserID(c.Query("user_id"))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		userId = parsedUserId
	}

	statements, total, err := model.ListCustomerMonthlyStatements(userId, c.Query("bill_month"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(statements)
	common.ApiSuccess(c, pageInfo)
}

func ExportAdminMonthlyStatementCSV(c *gin.Context) {
	userId, err := parseStatementUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := model.GenerateCustomerMonthlyStatement(userId, c.Query("bill_month"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeMonthlyStatementCSV(c, statement, 0)
}

func GetAdminBillingOverview(c *gin.Context) {
	overview, err := model.BuildAdminBillingOverview(c.Query("bill_month"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"overview": overview,
		"permissions": AdminBillingPermissions{
			CanAdjustGiftQuota:       model.HasPermission(c.GetInt("role"), c.GetString("staff_role"), common.PermissionFinanceWrite),
			CanAdjustPaidQuota:       model.HasPermission(c.GetInt("role"), c.GetString("staff_role"), common.PermissionSystemManage),
			CanEditFinancialSettings: model.HasPermission(c.GetInt("role"), c.GetString("staff_role"), common.PermissionFinanceSettings),
		},
	})
}

func writeMonthlyStatementCSV(c *gin.Context, statement *model.CustomerMonthlyStatement, userId int) {
	content, filename, err := model.ExportCustomerMonthlyStatementCSV(statement, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"statement.csv\"; filename*=UTF-8''"+url.PathEscape(filename))
	c.Data(200, "text/csv; charset=utf-8", content)
}

func parseStatementUserID(raw string) (int, error) {
	userId, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || userId <= 0 {
		return 0, errors.New("invalid user_id")
	}
	return userId, nil
}
