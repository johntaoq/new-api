package controller

import (
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GenerateChannelMonthlyBilling(c *gin.Context) {
	statement, err := model.GenerateChannelMonthlyStatement(c.Query("bill_month"), true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	bundle, err := model.BuildChannelMonthlyStatementBundle(statement, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func GetChannelMonthlyBilling(c *gin.Context) {
	statement, err := model.GenerateChannelMonthlyStatement(c.Query("bill_month"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	bundle, err := model.BuildChannelMonthlyStatementBundle(statement, common.GetPageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bundle)
}

func ListChannelMonthlyBillings(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	statements, total, err := model.ListChannelMonthlyStatements(c.Query("bill_month"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(statements)
	common.ApiSuccess(c, pageInfo)
}

func ExportChannelMonthlyBillingCSV(c *gin.Context) {
	statement, err := model.GenerateChannelMonthlyStatement(c.Query("bill_month"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	content, filename, err := model.ExportChannelMonthlyStatementCSV(statement)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"channel-billing.csv\"; filename*=UTF-8''"+url.PathEscape(filename))
	c.Data(200, "text/csv; charset=utf-8", content)
}
