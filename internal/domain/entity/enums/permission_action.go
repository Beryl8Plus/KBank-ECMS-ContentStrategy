package enums

// PermissionSource identifies the resource a permission applies to.
const (
	PermissionSourceDecisionRule = "decision_rule"
)

// PermissionAction identifies the operation a permission grants.
type PermissionAction string

const (
	PermissionActionCreate    PermissionAction = "CREATE"
	PermissionActionEdit      PermissionAction = "EDIT"
	PermissionActionDelete    PermissionAction = "DELETE"
	PermissionActionViewAll   PermissionAction = "VIEW_ALL"
	PermissionActionEditAll   PermissionAction = "EDIT_ALL"
	PermissionActionDeleteAll PermissionAction = "DELETE_ALL"
)
