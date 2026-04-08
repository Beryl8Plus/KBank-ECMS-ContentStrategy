package entity

// AllModels returns all GORM models in dependency order for AutoMigrate.
// Independent tables are listed first, followed by tables with foreign key dependencies.
func AllModels() []interface{} {
	return []interface{}{
		// 1. Independent tables (no FK deps)
		&Role{},
		&Profile{},
		&Placement{},
		&MDPSchemaRegistry{},
		&Calendar{},
		// 2. Tables with FK to independent tables
		&User{},
		&Permission{},
		&Attribute{},
		&LoginTokenHistory{},
		&CalendarDate{},
		// 3. Junction / dependent tables
		&ProfilePermission{},
		&DecisionRule{},
		&Rule{},
		&RuleCondition{},
		&Schedule{},
		&ScheduleOccurrence{},
	}
}
