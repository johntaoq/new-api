package common

import "strings"

const (
	StaffRoleNone    = ""
	StaffRoleAdmin   = "admin"
	StaffRoleFinance = "finance"
	StaffRoleAudit   = "audit"
	StaffRoleRoot    = "root"
)

const (
	PermissionOpsManage        = "ops.manage"
	PermissionFinanceView      = "finance.view"
	PermissionFinanceWrite     = "finance.write"
	PermissionFinanceAuditView = "finance.audit.view"
	PermissionFinanceSettings  = "finance.settings"
	PermissionSystemManage     = "system.manage"
)

func NormalizeStaffRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case StaffRoleAdmin:
		return StaffRoleAdmin
	case StaffRoleFinance:
		return StaffRoleFinance
	case StaffRoleAudit:
		return StaffRoleAudit
	case StaffRoleRoot:
		return StaffRoleRoot
	default:
		return StaffRoleNone
	}
}

func IsValidStaffRole(role string) bool {
	return NormalizeStaffRole(role) != StaffRoleNone || strings.TrimSpace(role) == ""
}
