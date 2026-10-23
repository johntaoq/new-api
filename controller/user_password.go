package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ResetManagedUserPassword is separate from profile updates and uses the
// administrator's authenticated session, not the managed user's credentials.
func ResetManagedUserPassword(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	var req struct {
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
		Confirmed       bool   `json:"confirmed"`
	}
	if err != nil || id <= 0 || common.DecodeJson(c.Request.Body, &req) != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if !req.Confirmed || req.Password != req.ConfirmPassword {
		common.ApiErrorT(c, "Password confirmation does not match")
		return
	}
	if err := model.ResetManagedUserPassword(c.GetInt("id"), id, req.Password); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, id, "user.password_reset", map[string]any{
		"id": id, "password_reset": true,
	})
	common.ApiSuccessT(c, "Password reset successfully", nil)
}
