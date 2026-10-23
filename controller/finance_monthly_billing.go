package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetFinanceMonthlyBilling(c *gin.Context) {
	months, err := model.ListFinanceBillingMonths()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
	next := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	common.ApiSuccess(c, gin.H{
		"months": months, "current_month": now.Format("2006-01"), "next_month": next.Format("2006-01"),
		"timezone": "Asia/Shanghai", "schedule": "01 00:30",
	})
}

func SetFinanceMonthlyRate(c *gin.Context) {
	var request struct {
		BillMonth string  `json:"bill_month"`
		Rate      float64 `json:"rate"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorT(c, "Invalid monthly exchange rate")
		return
	}
	if err := model.SetFinanceNextMonthRate(request.BillMonth, request.Rate, c.GetInt("id"), time.Now()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetFixedFinanceBills(c *gin.Context) {
	page, err := model.ListFixedFinanceBills(c.Query("bill_month"), c.Query("user_keyword"), 0, getFinancePageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetSelfFixedFinanceBills(c *gin.Context) {
	if c.GetInt("id") <= 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	page, err := model.ListFixedFinanceBills(c.Query("bill_month"), "", c.GetInt("id"), getFinancePageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

func GetFixedFinanceBillItems(c *gin.Context) {
	userID, err := parseRequiredFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	getFixedFinanceBillItems(c, userID)
}

func GetSelfFixedFinanceBillItems(c *gin.Context) {
	if c.GetInt("id") <= 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	getFixedFinanceBillItems(c, c.GetInt("id"))
}

func getFixedFinanceBillItems(c *gin.Context, userID int) {
	items, err := model.ListFixedFinanceBillItems(c.Query("bill_month"), userID, getFinancePageQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func ExportFixedFinanceBill(c *gin.Context) {
	userID, err := parseRequiredFinanceUserID(c.Query("user_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	exportFixedFinanceBill(c, userID)
}

func ExportSelfFixedFinanceBill(c *gin.Context) {
	if c.GetInt("id") <= 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	exportFixedFinanceBill(c, c.GetInt("id"))
}

func GetSelfCurrentFinanceBill(c *gin.Context) {
	bill, items, err := model.CurrentFinanceBill(c.GetInt("id"), time.Now())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page := getFinancePageQuery(c)
	start := min(page.GetStartIdx(), len(items))
	end := min(start+page.GetPageSize(), len(items))
	page.SetTotal(len(items))
	page.SetItems(items[start:end])
	common.ApiSuccess(c, gin.H{
		"items": items[start:end], "total": len(items), "page": page.Page, "page_size": page.GetPageSize(),
		"bill_month": bill.BillMonth, "cny_rate": bill.CNYRate, "as_of": bill.GeneratedAt,
	})
}

func ExportSelfCurrentFinanceBill(c *gin.Context) {
	now := time.Now()
	content, err := model.ExportCurrentFinanceBill(c.GetInt("id"), now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	month := now.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"current-details-%s-CNY.csv\"", month))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", content)
}

func exportFixedFinanceBill(c *gin.Context, userID int) {
	content, err := model.ExportFixedFinanceBill(c.Query("bill_month"), userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"monthly-bill-%d-%s.csv\"", userID, c.Query("bill_month")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", content)
}

// The legacy manual-generation URL now resumes due months only, never regenerates.
func ResumeFinanceMonthlyBilling(c *gin.Context) {
	months, err := model.ListFinanceBillingMonths()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	due := false
	for _, month := range months {
		if month.BillMonth == c.Query("bill_month") && month.ClosesAt <= time.Now().Unix() {
			due = true
			break
		}
	}
	if !due {
		common.ApiError(c, model.ErrFinanceMonthNotClosed)
		return
	}
	if err := model.RunFinanceMonthlyBilling(time.Now()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
