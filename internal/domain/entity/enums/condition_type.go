package enums

// ConditionType distinguishes a leaf condition from a group container row.
type ConditionType string

const (
	ConditionTypeCondition ConditionType = "condition"
	ConditionTypeGroup     ConditionType = "group"
)

func (t ConditionType) IsValid() bool {
	return t == ConditionTypeCondition || t == ConditionTypeGroup
}
