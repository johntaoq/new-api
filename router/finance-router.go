package router

import (
	"net/http"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func setFinanceRoutes(api *gin.RouterGroup) {
	handlePermissionRoute(api.Group("", middleware.AdminAuth()), http.MethodPost,
		"/user/:id/quota_adjust", authz.FinanceWrite, controller.AdjustUserWallet)
	self := api.Group("/finance/self", middleware.DisableCache(), middleware.UserAuth())
	self.GET("/statements", controller.GetSelfFixedFinanceBills)
	self.GET("/statements/details", controller.GetSelfFixedFinanceBillItems)
	self.GET("/statements/export", controller.ExportSelfFixedFinanceBill)
	self.GET("/current", controller.GetSelfCurrentFinanceBill)
	self.GET("/current/export", controller.ExportSelfCurrentFinanceBill)
	group := api.Group("/finance", middleware.DisableCache(), middleware.AdminAuth())
	for _, route := range []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{"/dashboard/summary", controller.GetFinanceDashboardSummary},
		{"/dashboard/todos", controller.GetFinanceDashboardTodos},
		{"/dashboard/rankings", controller.GetFinanceDashboardRankings},
		{"/revenue/summary", controller.GetFinanceRevenueSummary},
		{"/revenue/paid-sources", controller.GetFinancePaidSources},
		{"/revenue/gift-audit-summary", controller.GetFinanceGiftAuditSummary},
		{"/revenue/gift-audit", controller.GetFinanceGiftAudit},
		{"/channel-cost/summary", controller.GetFinanceChannelCostSummary},
		{"/channel-cost/channels", controller.GetFinanceChannelCostChannels},
		{"/channel-cost/models", controller.GetFinanceChannelCostModels},
		{"/channel-cost/channels/export", controller.ExportFinanceChannelCostCSV},
		{"/channel-cost/models/export", controller.ExportFinanceChannelCostCSV},
		{"/customer-bills/summary", controller.GetFixedFinanceBills},
		{"/customer-bills/details", controller.GetFixedFinanceBillItems},
		{"/customer-bills/export", controller.ExportFixedFinanceBill},
		{"/monthly-billing", controller.GetFinanceMonthlyBilling},
	} {
		handlePermissionRoute(group, http.MethodGet, route.path, authz.FinanceRead, route.handler)
	}
	handlePermissionRoute(group, http.MethodPost, "/customer-bills/generate", authz.FinanceWrite, controller.ResumeFinanceMonthlyBilling)
	handlePermissionRoute(group, http.MethodPut, "/monthly-billing/rate", authz.FinanceWrite, controller.SetFinanceMonthlyRate)
	handlePermissionRoute(group, http.MethodGet, "/audit/logs", authz.FinanceAudit, controller.GetFinanceAuditLogs)
}
