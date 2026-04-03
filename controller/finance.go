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

func GetFinanceDashboardSummary(c *gin.Context) {
	summary, err := model.BuildFinanceDashboardSummary(c.Query("period_type"), c.Query("period"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetFinanceDashboardTodos(c *gin.Context) {
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := model.ListFinanceDashboardTodos(c.Query("period_type"), c.Query("period"), limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

func GetFinanceDashboardRankings(c *gin.Context) {
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := model.ListFinanceDashboardRankings(c.Query("period_type"), c.Query("period"), c.Query("view"), limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

func GetFinanceRevenueSummary(c *gin.Context) {
	summary, err := model.BuildFinanceRevenueSummary(c.Query("period_type"), c.Query("period"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetFinancePaidSources(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	view := strings.TrimSpace(c.Query("view"))
	if view == "detail" {
		result, err := model.ListFinancePaidSourceDetails(c.Query("period_type"), c.Query("period"), pageInfo)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, result)
		return
	}
	result, err := model.ListFinancePaidSourceSummary(c.Query("period_type"), c.Query("period"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceGiftAudit(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID, err := parseOptionalFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := model.ListFinanceGiftAudit(c.Query("period_type"), c.Query("period"), c.Query("source_type"), userID, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceGiftAuditSummary(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID, err := parseOptionalFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := model.ListFinanceGiftAuditSummary(c.Query("period_type"), c.Query("period"), c.Query("source_type"), userID, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceChannelCostSummary(c *gin.Context) {
	summary, err := model.BuildFinanceChannelCostSummary(c.Query("period_type"), c.Query("period"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetFinanceChannelCostChannels(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	result, err := model.ListFinanceChannelCostChannels(c.Query("period_type"), c.Query("period"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceChannelCostModels(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	result, err := model.ListFinanceChannelCostModels(c.Query("period_type"), c.Query("period"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceCustomerBillSummary(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	result, err := model.ListFinanceCustomerBillSummaryV2(c.Query("bill_month"), c.Query("user_keyword"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GenerateFinanceCustomerBills(c *gin.Context) {
	result, err := model.GenerateFinanceCustomerBills(c.Query("bill_month"), c.Query("user_keyword"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetFinanceCustomerBillDetails(c *gin.Context) {
	userID, err := parseRequiredFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items, err := model.GetFinanceCustomerBillDetailsV2(c.Query("bill_month"), userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

func ExportFinanceCustomerBillCSV(c *gin.Context) {
	userID, err := parseRequiredFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	content, filename, err := model.ExportFinanceCustomerBillCSVV2(c.Query("bill_month"), userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"customer-bill.csv\"; filename*=UTF-8''"+url.PathEscape(filename))
	c.Data(200, "text/csv; charset=utf-8", content)
}

func GetFinanceAuditLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	result, err := model.ListFinanceAuditLogs(
		c.Query("period_type"),
		c.Query("period"),
		c.Query("module"),
		c.Query("action"),
		c.Query("operator_keyword"),
		c.Query("target_keyword"),
		pageInfo,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func parseRequiredFinanceUserID(raw string) (int, error) {
	userID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || userID <= 0 {
		return 0, errors.New("invalid user_id")
	}
	return userID, nil
}

func parseOptionalFinanceUserID(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return parseRequiredFinanceUserID(raw)
}
