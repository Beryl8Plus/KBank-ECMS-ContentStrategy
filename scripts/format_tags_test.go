package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatTagsPreservesEscapedQuotes(t *testing.T) {
	tag := "`gorm:\"size:255;index:idx_decision_rules_active_status,where:\\\"STATUS\\\" = 'ACTIVE' AND \\\"DELETED_AT\\\" IS NULL\" json:\"status\"`"

	formatted := formatTags(tag, len("gorm:\"size:255;index:idx_decision_rules_active_status,where:\\\"STATUS\\\" = 'ACTIVE' AND \\\"DELETED_AT\\\" IS NULL\""))

	if formatted != tag {
		t.Fatalf("formatted tag mismatch\nwant: %s\ngot:  %s", tag, formatted)
	}
}

func TestProcessFilePreservesIndexValue(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.go")
	content := `package sample

type DecisionRule struct {
	Status string ` + "`gorm:\"size:255;index:idx_decision_rules_active_status,where:\\\"STATUS\\\" = 'ACTIVE' AND \\\"DELETED_AT\\\" IS NULL\" json:\"status\"`" + `
	Name   string ` + "`gorm:\"size:255\" json:\"name\"`" + `
}
`

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	if err := processFile(filePath); err != nil {
		t.Fatalf("process file: %v", err)
	}

	formatted, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read formatted file: %v", err)
	}

	got := string(formatted)
	if !strings.Contains(got, `idx_decision_rules_active_status`) {
		t.Fatalf("formatted file lost index name: %s", got)
	}
	if !strings.Contains(got, `where:\"STATUS\" = 'ACTIVE' AND \"DELETED_AT\" IS NULL`) {
		t.Fatalf("formatted file lost where clause: %s", got)
	}
}
