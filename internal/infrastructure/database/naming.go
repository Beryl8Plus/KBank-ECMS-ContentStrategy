package database

import (
	"strings"

	"gorm.io/gorm/schema"
)

// UpperSnakeColumnNamingStrategy preserves the existing table names while
// normalizing auto-migrated column names to CAPITAL_SNAKE_CASE.
type UpperSnakeColumnNamingStrategy struct {
	schema.NamingStrategy
}

func (n UpperSnakeColumnNamingStrategy) ColumnName(table, column string) string {
	return strings.ToUpper(n.NamingStrategy.ColumnName(table, column))
}
