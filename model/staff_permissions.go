package model

import "github.com/QuantumNous/new-api/common"

type PermissionSet map[string]bool

func ResolveEffectiveStaffRole(userRole int, staffRole string) string {
	if userRole >= common.RoleRootUser {
		return common.StaffRoleRoot
	}
	normalized := common.NormalizeStaffRole(staffRole)
	if normalized != common.StaffRoleNone {
		return normalized
	}
	if userRole >= common.RoleAdminUser {
		return common.StaffRoleAdmin
	}
	return common.StaffRoleNone
}

func ResolvePermissionSet(userRole int, staffRole string) PermissionSet {
	permissions := PermissionSet{}
	effectiveRole := ResolveEffectiveStaffRole(userRole, staffRole)

	if effectiveRole == common.StaffRoleRoot {
		permissions[common.PermissionOpsManage] = true
		permissions[common.PermissionFinanceView] = true
		permissions[common.PermissionFinanceWrite] = true
		permissions[common.PermissionFinanceAuditView] = true
		permissions[common.PermissionFinanceSettings] = true
		permissions[common.PermissionSystemManage] = true
		return permissions
	}

	switch effectiveRole {
	case common.StaffRoleAdmin:
		permissions[common.PermissionOpsManage] = true
	case common.StaffRoleFinance:
		permissions[common.PermissionFinanceView] = true
		permissions[common.PermissionFinanceWrite] = true
	case common.StaffRoleAudit:
		permissions[common.PermissionFinanceView] = true
		permissions[common.PermissionFinanceAuditView] = true
	}
	return permissions
}

func HasPermission(userRole int, staffRole string, permission string) bool {
	return ResolvePermissionSet(userRole, staffRole)[permission]
}

func HasAnyPermission(userRole int, staffRole string, permissions ...string) bool {
	set := ResolvePermissionSet(userRole, staffRole)
	for _, permission := range permissions {
		if set[permission] {
			return true
		}
	}
	return false
}

func IsPrivilegedUser(user *User) bool {
	if user == nil {
		return false
	}
	return ResolveEffectiveStaffRole(user.Role, user.StaffRole) != common.StaffRoleNone
}

func CanViewUserForFinance(userRole int, staffRole string, target *User) bool {
	if target == nil {
		return false
	}
	if HasPermission(userRole, staffRole, common.PermissionSystemManage) {
		return true
	}
	if !HasAnyPermission(userRole, staffRole, common.PermissionFinanceView, common.PermissionFinanceWrite) {
		return false
	}
	if target.Role >= common.RoleRootUser || IsPrivilegedUser(target) {
		return false
	}
	return true
}

func CanAdjustUserFinance(userRole int, staffRole string, target *User) bool {
	if target == nil {
		return false
	}
	if HasPermission(userRole, staffRole, common.PermissionSystemManage) {
		return target.Role < common.RoleRootUser
	}
	if !HasPermission(userRole, staffRole, common.PermissionFinanceWrite) {
		return false
	}
	if target.Role >= common.RoleRootUser || IsPrivilegedUser(target) {
		return false
	}
	return true
}

func CanManageOpsTarget(userRole int, staffRole string, target *User) bool {
	if target == nil {
		return false
	}
	if HasPermission(userRole, staffRole, common.PermissionSystemManage) {
		return true
	}
	if !HasPermission(userRole, staffRole, common.PermissionOpsManage) {
		return false
	}
	if target.Role >= common.RoleRootUser || IsPrivilegedUser(target) {
		return false
	}
	return true
}

func BuildSidebarPermissionModules(userRole int, staffRole string) map[string]interface{} {
	modules := map[string]interface{}{
		"chat": map[string]interface{}{
			"enabled":          true,
			"playground":       true,
			"image_playground": true,
			"chat":             true,
		},
		"console": map[string]interface{}{
			"enabled":    true,
			"detail":     true,
			"token":      true,
			"log":        true,
			"midjourney": true,
			"task":       true,
		},
		"personal": map[string]interface{}{
			"enabled":  true,
			"topup":    true,
			"personal": true,
		},
	}

	adminModules := map[string]interface{}{
		"enabled":      false,
		"billing":      false,
		"channel":      false,
		"models":       false,
		"deployment":   false,
		"redemption":   false,
		"user":         false,
		"subscription": false,
		"setting":      false,
	}

	if HasPermission(userRole, staffRole, common.PermissionFinanceView) {
		adminModules["enabled"] = true
		adminModules["billing"] = true
	}
	if HasPermission(userRole, staffRole, common.PermissionFinanceWrite) {
		adminModules["enabled"] = true
		adminModules["redemption"] = true
		adminModules["user"] = true
	}
	if HasPermission(userRole, staffRole, common.PermissionOpsManage) {
		adminModules["enabled"] = true
		adminModules["channel"] = true
		adminModules["models"] = true
		adminModules["deployment"] = true
		adminModules["user"] = true
		adminModules["subscription"] = true
	}
	if HasPermission(userRole, staffRole, common.PermissionSystemManage) {
		adminModules["enabled"] = true
		adminModules["setting"] = true
	}

	modules["admin"] = adminModules
	return modules
}
