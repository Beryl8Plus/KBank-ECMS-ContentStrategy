// Command mockdata generates goose SQL mock data for decision rules.
//
// Usage:
//
//	go run ./scripts/mockdata
//	  Reuse the latest cmd/migrate/mocks/*_decision_rule_example_data.sql file.
//	  If no file exists yet, run make db-mock-create-sql name=decision_rule_example_data
//	  and write the generated SQL into that new file.
//
//	go run ./scripts/mockdata -name=decision_rule_exaple_data
//	  Create or reuse the latest goose mock file matching the provided migration name.
//	  Use this when the target filename already follows a different naming convention.
//
//	go run ./scripts/mockdata -out=cmd/migrate/mocks/20260417000000_custom.sql
//	  Write directly to the given path and skip the Makefile lookup/create flow.
//
//	go run ./scripts/mockdata -count=100
//	  Change the number of generated mock rule sets.
//
// Notes:
//   - The random seed is based on the current UTC date at 00:00:00, so reruns on the
//     same day produce the same dataset.
//   - Change the seed logic in main() if you need a different repeatability strategy.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

const (
	defaultMockDir  = "cmd/migrate/mocks"
	defaultMockName = "decision_rule_example_data"
)

type attributeDef struct {
	ID           string
	FieldName    string
	DisplayName  string
	DataType     string
	ValueSQL     string
	Description  string
	SourceSystem string
	IsActive     bool
}

type mockSet struct {
	PlacementID          string
	PlacementName        string
	PlacementDescription string
	MaxResults           int
	DecisionRuleID       string
	DecisionRuleName     string
	ContentPath          string
	DecisionScore        float64
	ConditionID          string
	AttributeID          string
	LogicalOperator      string
	ConnectorOperator    string
	RuleID               string
	VariationName        string
	RuleScore            float64
	OrderNo              int
	RuleAttributeID      string
	RuleValueSQL         string
	ScheduleID           string
	EffectiveFromExpr    string
	EffectiveUntilExpr   string
}

func main() {
	outPath := flag.String("out", "", "output SQL file path")
	mockName := flag.String("name", defaultMockName, "mock migration name used with make db-mock-create-sql")
	count := flag.Int("count", 50, "number of mock decision rule sets")
	flag.Parse()

	if *count <= 0 {
		fail("count must be greater than zero")
	}

	resolvedOutputPath := *outPath
	if resolvedOutputPath == "" {
		resolvedOutputPath = resolveOutputPath(*mockName)
	}

	// Seed with date to ensure consistent mock data generation across runs. Change seed value to generate different mock data.
	timeSeed := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Unix()
	_ = gofakeit.Seed(timeSeed)

	schemaID := gofakeit.UUID()
	attributes := buildAttributes(schemaID)
	mockSets := buildMockSets(*count, attributes)
	sql := buildSQL(timeSeed, schemaID, attributes, mockSets)

	if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
		fail("create output directory: %v", err)
	}

	if err := os.WriteFile(resolvedOutputPath, []byte(sql), 0o644); err != nil {
		fail("write output file: %v", err)
	}
}

func resolveOutputPath(mockName string) string {
	if mockName == "" {
		fail("name must not be empty")
	}

	if existingPath, ok := findLatestMockFile(mockName); ok {
		return existingPath
	}

	command := exec.Command("make", "db-mock-create-sql", fmt.Sprintf("name=%s", mockName))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fail("create mock SQL file via make: %v", err)
	}

	if createdPath, ok := findLatestMockFile(mockName); ok {
		return createdPath
	}

	fail("unable to locate generated mock SQL file for name %q", mockName)
	return ""
}

func findLatestMockFile(mockName string) (string, bool) {
	pattern := filepath.Join(defaultMockDir, fmt.Sprintf("*_%s.sql", mockName))
	paths, err := filepath.Glob(pattern)
	if err != nil {
		fail("glob mock SQL files: %v", err)
	}
	if len(paths) == 0 {
		return "", false
	}

	sort.Strings(paths)
	return paths[len(paths)-1], true
}

func buildAttributes(schemaID string) []attributeDef {
	return []attributeDef{
		{
			ID:           gofakeit.UUID(),
			FieldName:    "segment",
			DisplayName:  "User Segment",
			DataType:     "Text",
			ValueSQL:     `'["Mass","Affluent","VIP","Young Wealth","SME"]'`,
			Description:  "Customer segmentation bucket from CLEN",
			SourceSystem: "CLEN",
			IsActive:     true,
		},
		{
			ID:           gofakeit.UUID(),
			FieldName:    "user_age",
			DisplayName:  "User Age",
			DataType:     "Number",
			ValueSQL:     "NULL",
			Description:  "Age of the user in years",
			SourceSystem: "CLEN",
			IsActive:     true,
		},
		{
			ID:           gofakeit.UUID(),
			FieldName:    "region",
			DisplayName:  "Region",
			DataType:     "Text",
			ValueSQL:     `'["Bangkok","Central","North","Northeast","South"]'`,
			Description:  "Preferred operating region of the user",
			SourceSystem: "CLEN",
			IsActive:     true,
		},
		{
			ID:           gofakeit.UUID(),
			FieldName:    "risk_level",
			DisplayName:  "Risk Level",
			DataType:     "Number",
			ValueSQL:     `'[1,2,3,4,5]'`,
			Description:  "Investment risk level score",
			SourceSystem: "CLEN",
			IsActive:     true,
		},
	}
}

func buildMockSets(count int, attributes []attributeDef) []mockSet {
	channels := []string{"Home", "Portfolio", "Insight", "Offer", "Wealth", "Deposit", "Credit", "Reward"}
	surfaces := []string{"Banner", "Carousel", "Tile", "Popup", "Widget", "Spotlight"}
	zones := []string{"Hero", "Top", "Middle", "Bottom", "Sidebar"}
	variationPrefixes := []string{"Prime", "Growth", "Focus", "Priority", "Select", "Momentum"}
	segments := []string{"Mass", "Affluent", "VIP", "Young Wealth", "SME"}
	regions := []string{"Bangkok", "Central", "North", "Northeast", "South"}

	sets := make([]mockSet, 0, count)
	for index := 1; index <= count; index++ {
		channel := gofakeit.RandomString(channels)
		surface := gofakeit.RandomString(surfaces)
		zone := gofakeit.RandomString(zones)
		prefix := gofakeit.RandomString(variationPrefixes)
		attribute := attributes[gofakeit.Number(0, len(attributes)-1)]

		logicalOperator, ruleValueSQL, conditionLabel := buildCondition(attribute.FieldName, segments, regions)
		decisionScore := round1(gofakeit.Float64Range(0.5, 8.5))
		ruleScore := round1(gofakeit.Float64Range(5, 30))
		maxResults := gofakeit.Number(1, 10)
		fromDaysAgo := gofakeit.Number(0, 45)
		untilDaysAhead := gofakeit.Number(90, 365)

		// Use a fixed set of placement names (4 items only)
		placementOptions := []string{"wsaHomeBanner", "wsaPortBanner", "wsaSplash", "wsaLandingPage"}
		placementName := gofakeit.RandomString(placementOptions)
		sets = append(sets, mockSet{
			PlacementID:          gofakeit.UUID(),
			PlacementName:        placementName,
			PlacementDescription: fmt.Sprintf("%s %s placement for %s zone", channel, strings.ToLower(surface), strings.ToLower(zone)),
			MaxResults:           maxResults,
			DecisionRuleID:       gofakeit.UUID(),
			DecisionRuleName:     placementName,
			ContentPath:          fmt.Sprintf("personalizedContent/%s/%s-%02d", strings.ToLower(channel), strings.ToLower(surface), index),
			DecisionScore:        decisionScore,
			ConditionID:          gofakeit.UUID(),
			AttributeID:          attribute.ID,
			LogicalOperator:      logicalOperator,
			ConnectorOperator:    "AND",
			RuleID:               gofakeit.UUID(),
			VariationName:        fmt.Sprintf("%s %s %02d", prefix, conditionLabel, index),
			RuleScore:            ruleScore,
			OrderNo:              1,
			RuleAttributeID:      gofakeit.UUID(),
			RuleValueSQL:         ruleValueSQL,
			ScheduleID:           gofakeit.UUID(),
			EffectiveFromExpr:    fmt.Sprintf("NOW() - interval '%d day'", fromDaysAgo),
			EffectiveUntilExpr:   fmt.Sprintf("NOW() + interval '%d day'", untilDaysAhead),
		})
	}

	return sets
}

func buildCondition(fieldName string, segments []string, regions []string) (string, string, string) {
	switch fieldName {
	case "segment":
		value := gofakeit.RandomString(segments)
		return "=", sqlStringLiteral(fmt.Sprintf("\"%s\"", value)), fmt.Sprintf("Segment %s", value)
	case "user_age":
		operator := gofakeit.RandomString([]string{"<=", ">="})
		value := gofakeit.Number(20, 65)
		return operator, fmt.Sprintf("'%d'", value), fmt.Sprintf("Age %s %d", operator, value)
	case "region":
		value := gofakeit.RandomString(regions)
		return "=", sqlStringLiteral(fmt.Sprintf("\"%s\"", value)), fmt.Sprintf("Region %s", value)
	case "risk_level":
		operator := gofakeit.RandomString([]string{"<=", ">="})
		value := gofakeit.Number(1, 5)
		return operator, fmt.Sprintf("'%d'", value), fmt.Sprintf("Risk %s %d", operator, value)
	default:
		return "=", sqlStringLiteral("\"Mass\""), "Segment Mass"
	}
}

func buildSQL(seed int64, schemaID string, attributes []attributeDef, sets []mockSet) string {
	var buf bytes.Buffer

	buf.WriteString("-- +goose Up\n")
	buf.WriteString("-- Generated by scripts/generate_decision_rule_mock.go\n")
	buf.WriteString(fmt.Sprintf("-- Seed: %d\n\n", seed))

	writeInsert(&buf, "placements",
		[]string{"\"ID\"", "\"NAME\"", "\"DESCRIPTION\"", "\"MAX_RESULTS\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		placementValues(sets),
	)

	writeInsert(&buf, "clen_schema_registry",
		[]string{"\"ID\"", "\"SCHEMA_NAME\"", "\"VERSION\"", "\"SCHEMA_DEFINITION\"", "\"IS_ACTIVE\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		[][]string{{
			sqlStringLiteral(schemaID),
			sqlStringLiteral("PersonalizationAudience"),
			sqlStringLiteral("1.0.0"),
			sqlStringLiteral(`{"type":"object","properties":{"segment":{"type":"string"},"user_age":{"type":"number"},"region":{"type":"string"},"risk_level":{"type":"number"}}}`),
			"true",
			"NOW()",
			"NOW()",
		}},
	)

	writeInsert(&buf, "attributes",
		[]string{"\"ID\"", "\"CLEN_SCHEMA_REGISTRY_ID\"", "\"FIELD_NAME\"", "\"DISPLAY_NAME\"", "\"DATA_TYPE\"", "\"VALUE\"", "\"DESCRIPTION\"", "\"SOURCE_SYSTEM\"", "\"IS_ACTIVE\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		attributeValues(schemaID, attributes),
	)

	writeInsert(&buf, "decision_rules",
		[]string{"\"ID\"", "\"NAME\"", "\"TYPE\"", "\"EVALUATE_TYPE\"", "\"CONTENT_PATH\"", "\"SCORE\"", "\"STATUS\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		decisionRuleValues(sets),
	)

	writeInsert(&buf, "rule_conditions",
		[]string{"\"ID\"", "\"SEQUENCE\"", "\"DECISION_RULE_ID\"", "\"ATTRIBUTE_ID\"", "\"LOGICAL_OPERATOR\"", "\"CONNECTOR_OPERATOR\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		ruleConditionValues(sets),
	)

	writeInsert(&buf, "rules",
		[]string{"\"ID\"", "\"DECISION_RULE_ID\"", "\"VARIATION_NAME\"", "\"SCORE\"", "\"ORDER_NO\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		ruleValues(sets),
	)

	writeInsert(&buf, "rule_attributes",
		[]string{"\"ID\"", "\"RULE_ID\"", "\"ATTRIBUTE_ID\"", "\"VALUE\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		ruleAttributeValues(sets),
	)

	writeInsert(&buf, "schedules",
		[]string{"\"ID\"", "\"DECISION_RULE_ID\"", "\"PLACEMENT_ID\"", "\"RECURRENCE_TYPE\"", "\"EFFECTIVE_FROM\"", "\"EFFECTIVE_UNTIL\"", "\"IS_ACTIVE\"", "\"CREATED_AT\"", "\"UPDATED_AT\""},
		scheduleValues(sets),
	)

	buf.WriteString("-- +goose Down\n")
	writeDelete(&buf, "schedules", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.ScheduleID }))
	writeDelete(&buf, "rule_attributes", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.RuleAttributeID }))
	writeDelete(&buf, "rules", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.RuleID }))
	writeDelete(&buf, "rule_conditions", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.ConditionID }))
	writeDelete(&buf, "decision_rules", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.DecisionRuleID }))
	writeDelete(&buf, "attributes", "\"ID\"", collectAttributeIDs(attributes))
	buf.WriteString(fmt.Sprintf("DELETE FROM clen_schema_registry WHERE \"ID\" = %s;\n", sqlStringLiteral(schemaID)))
	writeDelete(&buf, "placements", "\"ID\"", collectIDs(sets, func(item mockSet) string { return item.PlacementID }))

	return buf.String()
}

func placementValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.PlacementID),
			sqlStringLiteral(item.PlacementName),
			sqlStringLiteral(item.PlacementDescription),
			fmt.Sprintf("%d", item.MaxResults),
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func attributeValues(schemaID string, attributes []attributeDef) [][]string {
	values := make([][]string, 0, len(attributes))
	for _, item := range attributes {
		values = append(values, []string{
			sqlStringLiteral(item.ID),
			sqlStringLiteral(schemaID),
			sqlStringLiteral(item.FieldName),
			sqlStringLiteral(item.DisplayName),
			sqlStringLiteral(item.DataType),
			item.ValueSQL,
			sqlStringLiteral(item.Description),
			sqlStringLiteral(item.SourceSystem),
			fmt.Sprintf("%t", item.IsActive),
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func decisionRuleValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.DecisionRuleID),
			sqlStringLiteral(item.DecisionRuleName),
			sqlStringLiteral("AUDIENCE"),
			sqlStringLiteral("SCORING"),
			sqlStringLiteral(item.ContentPath),
			formatFloat(item.DecisionScore),
			sqlStringLiteral("ACTIVE"),
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func ruleConditionValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.ConditionID),
			"1",
			sqlStringLiteral(item.DecisionRuleID),
			sqlStringLiteral(item.AttributeID),
			sqlStringLiteral(item.LogicalOperator),
			sqlStringLiteral(item.ConnectorOperator),
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func ruleValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.RuleID),
			sqlStringLiteral(item.DecisionRuleID),
			sqlStringLiteral(item.VariationName),
			formatFloat(item.RuleScore),
			fmt.Sprintf("%d", item.OrderNo),
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func ruleAttributeValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.RuleAttributeID),
			sqlStringLiteral(item.RuleID),
			sqlStringLiteral(item.AttributeID),
			item.RuleValueSQL,
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func scheduleValues(sets []mockSet) [][]string {
	values := make([][]string, 0, len(sets))
	for _, item := range sets {
		values = append(values, []string{
			sqlStringLiteral(item.ScheduleID),
			sqlStringLiteral(item.DecisionRuleID),
			sqlStringLiteral(item.PlacementID),
			sqlStringLiteral("ONCE"),
			item.EffectiveFromExpr,
			item.EffectiveUntilExpr,
			"true",
			"NOW()",
			"NOW()",
		})
	}
	return values
}

func writeInsert(buf *bytes.Buffer, table string, columns []string, values [][]string) {
	buf.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES\n", table, strings.Join(columns, ", ")))
	for index, row := range values {
		separator := ","
		if index == len(values)-1 {
			separator = ";"
		}
		buf.WriteString(fmt.Sprintf("(%s)%s\n", strings.Join(row, ", "), separator))
	}
	buf.WriteString("\n")
}

func writeDelete(buf *bytes.Buffer, table string, column string, ids []string) {
	buf.WriteString(fmt.Sprintf("DELETE FROM %s WHERE %s IN (\n", table, column))
	for index, id := range ids {
		separator := ","
		if index == len(ids)-1 {
			separator = ""
		}
		buf.WriteString(fmt.Sprintf("    %s%s\n", sqlStringLiteral(id), separator))
	}
	buf.WriteString(");\n")
}

func collectAttributeIDs(attributes []attributeDef) []string {
	ids := make([]string, 0, len(attributes))
	for _, item := range attributes {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectIDs(items []mockSet, getter func(mockSet) string) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, getter(item))
	}
	return ids
}

func sqlStringLiteral(value string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "''"))
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.1f", value)
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func fail(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
