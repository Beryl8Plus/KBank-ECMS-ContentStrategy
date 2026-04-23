// Command mockdata generates goose SQL mock data for decision rules and a
// companion Redis seed shell script that uses the same attribute UUIDs.
//
// Usage:
//
//	go run ./scripts/mockdata
//	  Reuse the latest cmd/migrate/mocks/*_decision_rule_example_data.sql file.
//	  If no file exists yet, run make db-mock-create-sql name=decision_rule_example_data
//	  and write the generated SQL into that new file.
//	  Also writes scripts/seed-redis-user-attrs.sh with consistent attribute UUIDs.
//
//	go run ./scripts/mockdata -name=decision_rule_example_data
//	  Create or reuse the latest goose mock file matching the provided migration name.
//
//	go run ./scripts/mockdata -out=cmd/migrate/mocks/20260417000000_custom.sql
//	  Write directly to the given path and skip the Makefile lookup/create flow.
//
//	go run ./scripts/mockdata -count=100
//	  Change the number of generated mock rule sets.
//
//	go run ./scripts/mockdata -redis-out=scripts/seed-redis-user-attrs.sh
//	  Override the Redis seed script output path (default: scripts/seed-redis-user-attrs.sh).
//	  Pass an empty string to skip Redis seed generation.
//
// Notes:
//   - The random seed is based on the current UTC date at 00:00:00, so reruns on the
//     same day produce the same dataset.
//   - Change the seed logic in main() if you need a different repeatability strategy.
//   - The Redis seed script and the SQL file are always generated from the same UUID
//     pool, keeping attribute keys in sync across both outputs.
package main

import (
	"bytes"
	"encoding/json"
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
	defaultMockDir       = "cmd/migrate/mocks"
	defaultMockName      = "decision_rule_example_data"
	defaultRedisOut      = "scripts/seed-redis-user-attrs.sh"
	defaultUserSeedCount = 20
)

// channelDef holds channel identity data.
type channelDef struct {
	ID          string
	ChannelName string
}

// attributeDef describes one CLEN attribute.
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

// ruleVariation models one variation (rule row + rule_attribute row) within a decision rule.
type ruleVariation struct {
	RuleID          string
	VariationName   string
	RuleScore       float64
	OrderNo         int
	RuleAttributeID string
	AttributeID     string
	LogicalOperator string
	ValueSQL        string
}

// mockSet models one full scenario: channel → placement → decision rule → condition → N variations → schedule.
type mockSet struct {
	ChannelID          string
	PlacementID        string
	PlacementName      string
	DecisionRuleID     string
	DecisionRuleName   string
	ContentPath        string
	DecisionScore      float64
	ConditionID        string
	AttributeID        string
	LogicalOperator    string
	ConnectorOperator  string
	Variations         []ruleVariation
	ScheduleID         string
	EffectiveFromExpr  string
	EffectiveUntilExpr string
}

// userSeed describes one test user to seed into Redis.
type userSeed struct {
	CisID      string
	Segment    string
	UserAge    int
	Region     string
	RiskLevel  int
	Commentary string
}

func main() {
	outPath := flag.String("out", "", "output SQL file path")
	mockName := flag.String("name", defaultMockName, "mock migration name used with make db-mock-create-sql")
	count := flag.Int("count", 50, "number of mock decision rule sets")
	redisOut := flag.String("redis-out", defaultRedisOut, "Redis seed shell script output path; pass empty string to skip")
	flag.Parse()

	if *count <= 0 {
		fail("count must be greater than zero")
	}

	resolvedOutputPath := *outPath
	if resolvedOutputPath == "" {
		resolvedOutputPath = resolveOutputPath(*mockName)
	}

	// Seed with date to ensure consistent mock data generation across runs.
	timeSeed := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Unix()
	_ = gofakeit.Seed(timeSeed)

	schemaID := gofakeit.UUID()
	channels := buildChannels()
	attributes := buildAttributes()
	mockSets := buildMockSets(*count, channels, attributes)
	users := buildUsers(attributes)

	sqlContent := buildSQL(timeSeed, schemaID, channels, attributes, mockSets)
	if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
		fail("create output directory: %v", err)
	}
	if err := os.WriteFile(resolvedOutputPath, []byte(sqlContent), 0o644); err != nil {
		fail("write SQL output file: %v", err)
	}
	fmt.Printf("wrote SQL  → %s\n", resolvedOutputPath)

	if *redisOut != "" {
		redisScript := buildRedisSeedScript(attributes, users)
		if err := os.MkdirAll(filepath.Dir(*redisOut), 0o755); err != nil {
			fail("create redis-out directory: %v", err)
		}
		if err := os.WriteFile(*redisOut, []byte(redisScript), 0o755); err != nil {
			fail("write Redis seed script: %v", err)
		}
		fmt.Printf("wrote Redis seed → %s\n", *redisOut)
	}
}

// ---------------------------------------------------------------------------
// Output path resolution
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Data builders
// ---------------------------------------------------------------------------

func buildChannels() []channelDef {
	return []channelDef{
		{ID: gofakeit.UUID(), ChannelName: "Home"},
		{ID: gofakeit.UUID(), ChannelName: "Portfolio"},
		{ID: gofakeit.UUID(), ChannelName: "Wealth"},
		{ID: gofakeit.UUID(), ChannelName: "Offer"},
	}
}

func buildAttributes() []attributeDef {
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

func buildMockSets(count int, channels []channelDef, attributes []attributeDef) []mockSet {
	placementOptions := []string{"wsaHomeBanner", "wsaPortBanner", "wsaSplash", "wsaLandingPage"}
	variationPrefixes := []string{"Prime", "Growth", "Focus", "Priority", "Select", "Momentum", "Elite", "Core"}

	segments := []string{"Mass", "Affluent", "VIP", "Young Wealth", "SME"}
	regions := []string{"Bangkok", "Central", "North", "Northeast", "South"}

	sets := make([]mockSet, 0, count)
	for index := 1; index <= count; index++ {
		placementIdx := gofakeit.Number(0, len(placementOptions)-1)
		placementName := placementOptions[placementIdx]
		channel := channels[placementIdx]

		primaryAttr := attributes[gofakeit.Number(0, len(attributes)-1)]
		logicalOperator, _, _ := buildCondition(primaryAttr.FieldName, segments, regions)

		decisionScore := round1(gofakeit.Float64Range(0.5, 8.5))
		fromDaysAgo := gofakeit.Number(0, 45)
		untilDaysAhead := gofakeit.Number(90, 365)

		variationCount := gofakeit.Number(2, 4)
		variations := make([]ruleVariation, 0, variationCount)
		for v := 1; v <= variationCount; v++ {
			varAttr := attributes[gofakeit.Number(0, len(attributes)-1)]
			varOp, varValueSQL, varLabel := buildCondition(varAttr.FieldName, segments, regions)
			prefix := gofakeit.RandomString(variationPrefixes)
			variations = append(variations, ruleVariation{
				RuleID:          gofakeit.UUID(),
				VariationName:   fmt.Sprintf("%s %s %02d-%d", prefix, varLabel, index, v),
				RuleScore:       round1(gofakeit.Float64Range(5, 30)),
				OrderNo:         v,
				RuleAttributeID: gofakeit.UUID(),
				AttributeID:     varAttr.ID,
				LogicalOperator: varOp,
				ValueSQL:        varValueSQL,
			})
		}

		sets = append(sets, mockSet{
			ChannelID:          channel.ID,
			PlacementID:        gofakeit.UUID(),
			PlacementName:      placementName,
			DecisionRuleID:     gofakeit.UUID(),
			DecisionRuleName:   fmt.Sprintf("%s Rule %02d", placementName, index),
			ContentPath:        fmt.Sprintf("personalizedContent/%s/%s-%02d", strings.ToLower(channel.ChannelName), strings.ToLower(placementName), index),
			DecisionScore:      decisionScore,
			ConditionID:        gofakeit.UUID(),
			AttributeID:        primaryAttr.ID,
			LogicalOperator:    logicalOperator,
			ConnectorOperator:  "AND",
			Variations:         variations,
			ScheduleID:         gofakeit.UUID(),
			EffectiveFromExpr:  fmt.Sprintf("NOW() - interval '%d day'", fromDaysAgo),
			EffectiveUntilExpr: fmt.Sprintf("NOW() + interval '%d day'", untilDaysAhead),
		})
	}
	return sets
}

func buildCondition(fieldName string, segments, regions []string) (operator, valueSQL, label string) {
	switch fieldName {
	case "segment":
		v := gofakeit.RandomString(segments)
		return "=", sqlStringLiteral(fmt.Sprintf(`"%s"`, v)), fmt.Sprintf("Segment %s", v)
	case "user_age":
		op := gofakeit.RandomString([]string{"<=", ">="})
		v := gofakeit.Number(20, 65)
		return op, fmt.Sprintf("'%d'", v), fmt.Sprintf("Age %s %d", op, v)
	case "region":
		v := gofakeit.RandomString(regions)
		return "=", sqlStringLiteral(fmt.Sprintf(`"%s"`, v)), fmt.Sprintf("Region %s", v)
	case "risk_level":
		op := gofakeit.RandomString([]string{"<=", ">="})
		v := gofakeit.Number(1, 5)
		return op, fmt.Sprintf("'%d'", v), fmt.Sprintf("Risk %s %d", op, v)
	default:
		return "=", sqlStringLiteral(`"Mass"`), "Segment Mass"
	}
}

// buildUsers creates a deterministic set of test users that exercise all
// segment, region, age, and risk combinations relevant to the rules.
func buildUsers(attributes []attributeDef) []userSeed {
	attrByField := make(map[string]attributeDef, len(attributes))
	for _, a := range attributes {
		attrByField[a.FieldName] = a
	}

	return []userSeed{
		// Core segment × region matrix (one user per segment)
		{CisID: "cis-user-01", Segment: "Mass", UserAge: 28, Region: "Bangkok", RiskLevel: 2, Commentary: "Mass / Bangkok / age 28 / risk 2 — baseline mass user"},
		{CisID: "cis-user-02", Segment: "Affluent", UserAge: 35, Region: "Central", RiskLevel: 3, Commentary: "Affluent / Central / age 35 / risk 3 — mid-risk affluent"},
		{CisID: "cis-user-03", Segment: "VIP", UserAge: 45, Region: "North", RiskLevel: 4, Commentary: "VIP / North / age 45 / risk 4 — high-risk VIP"},
		{CisID: "cis-user-04", Segment: "Young Wealth", UserAge: 24, Region: "Northeast", RiskLevel: 1, Commentary: "Young Wealth / Northeast / age 24 / risk 1 — low-risk young"},
		{CisID: "cis-user-05", Segment: "SME", UserAge: 52, Region: "South", RiskLevel: 5, Commentary: "SME / South / age 52 / risk 5 — max-risk SME"},

		// Edge-case age boundaries
		{CisID: "cis-user-06", Segment: "Mass", UserAge: 60, Region: "Northeast", RiskLevel: 2, Commentary: "Mass / Northeast / age 60 / risk 2 — senior user triggering age >= rules"},
		{CisID: "cis-user-07", Segment: "Affluent", UserAge: 20, Region: "South", RiskLevel: 1, Commentary: "Affluent / South / age 20 / risk 1 — youngest legal user"},
		{CisID: "cis-user-08", Segment: "VIP", UserAge: 65, Region: "Bangkok", RiskLevel: 5, Commentary: "VIP / Bangkok / age 65 / risk 5 — oldest+max risk"},

		// Cross-segment risk extremes
		{CisID: "cis-user-09", Segment: "Young Wealth", UserAge: 22, Region: "Central", RiskLevel: 5, Commentary: "Young Wealth / Central / age 22 / risk 5 — young but high risk"},
		{CisID: "cis-user-10", Segment: "SME", UserAge: 48, Region: "North", RiskLevel: 1, Commentary: "SME / North / age 48 / risk 1 — risk-averse SME"},

		// Region coverage — Bangkok for each main segment
		{CisID: "cis-user-11", Segment: "Mass", UserAge: 33, Region: "Bangkok", RiskLevel: 3, Commentary: "Mass / Bangkok / age 33 / risk 3"},
		{CisID: "cis-user-12", Segment: "Affluent", UserAge: 40, Region: "Bangkok", RiskLevel: 4, Commentary: "Affluent / Bangkok / age 40 / risk 4"},
		{CisID: "cis-user-13", Segment: "VIP", UserAge: 55, Region: "Bangkok", RiskLevel: 2, Commentary: "VIP / Bangkok / age 55 / risk 2 — low-risk VIP"},

		// Region coverage — Central
		{CisID: "cis-user-14", Segment: "SME", UserAge: 42, Region: "Central", RiskLevel: 3, Commentary: "SME / Central / age 42 / risk 3"},
		{CisID: "cis-user-15", Segment: "Mass", UserAge: 50, Region: "Central", RiskLevel: 5, Commentary: "Mass / Central / age 50 / risk 5"},

		// Region coverage — North / Northeast / South
		{CisID: "cis-user-16", Segment: "Young Wealth", UserAge: 29, Region: "North", RiskLevel: 2, Commentary: "Young Wealth / North / age 29 / risk 2"},
		{CisID: "cis-user-17", Segment: "Affluent", UserAge: 38, Region: "Northeast", RiskLevel: 4, Commentary: "Affluent / Northeast / age 38 / risk 4"},
		{CisID: "cis-user-18", Segment: "VIP", UserAge: 47, Region: "South", RiskLevel: 3, Commentary: "VIP / South / age 47 / risk 3"},

		// Boundary age conditions
		{CisID: "cis-user-19", Segment: "Mass", UserAge: 30, Region: "Bangkok", RiskLevel: 3, Commentary: "Mass / Bangkok / age 30 / risk 3 — common age threshold"},
		{CisID: "cis-user-20", Segment: "SME", UserAge: 56, Region: "Northeast", RiskLevel: 4, Commentary: "SME / Northeast / age 56 / risk 4 — senior SME triggering age >= 56 rules"},
	}
}

// ---------------------------------------------------------------------------
// SQL builder
// ---------------------------------------------------------------------------

func buildSQL(seed int64, schemaID string, channels []channelDef, attributes []attributeDef, sets []mockSet) string {
	var buf bytes.Buffer

	buf.WriteString("-- +goose Up\n")
	buf.WriteString("-- Generated by scripts/mockdata/generate_decision_rule_mock.go\n")
	fmt.Fprintf(&buf, "-- Seed: %d\n\n", seed)

	writeInsert(&buf, "channels",
		[]string{`"ID"`, `"CHANNEL_NAME"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		channelValues(channels),
	)

	writeInsert(&buf, "placements",
		[]string{`"ID"`, `"PLACEMENT_NAME"`, `"CHANNEL_ID"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		placementValues(sets),
	)

	writeInsert(&buf, "clen_schema_registry",
		[]string{`"ID"`, `"SCHEMA_NAME"`, `"VERSION"`, `"SCHEMA_DEFINITION"`, `"IS_ACTIVE"`, `"CREATED_AT"`, `"UPDATED_AT"`},
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
		[]string{`"ID"`, `"CLEN_SCHEMA_REGISTRY_ID"`, `"FIELD_NAME"`, `"DISPLAY_NAME"`, `"DATA_TYPE"`, `"VALUE"`, `"DESCRIPTION"`, `"SOURCE_SYSTEM"`, `"IS_ACTIVE"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		attributeValues(schemaID, attributes),
	)

	writeInsert(&buf, "decision_rules",
		[]string{`"ID"`, `"NAME"`, `"TYPE"`, `"EVALUATE_TYPE"`, `"CONTENT_PATH"`, `"SCORE"`, `"STATUS"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		decisionRuleValues(sets),
	)

	writeInsert(&buf, "rule_conditions",
		[]string{`"ID"`, `"SEQUENCE"`, `"DECISION_RULE_ID"`, `"ATTRIBUTE_ID"`, `"LOGICAL_OPERATOR"`, `"CONNECTOR_OPERATOR"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		ruleConditionValues(sets),
	)

	writeInsert(&buf, "rules",
		[]string{`"ID"`, `"DECISION_RULE_ID"`, `"VARIATION_NAME"`, `"SCORE"`, `"ORDER_NO"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		ruleValues(sets),
	)

	writeInsert(&buf, "rule_attributes",
		[]string{`"ID"`, `"RULE_ID"`, `"ATTRIBUTE_ID"`, `"VALUE"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		ruleAttributeValues(sets),
	)

	writeInsert(&buf, "schedules",
		[]string{`"ID"`, `"DECISION_RULE_ID"`, `"PLACEMENT_ID"`, `"RECURRENCE_TYPE"`, `"EFFECTIVE_FROM"`, `"EFFECTIVE_UNTIL"`, `"IS_ACTIVE"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		scheduleValues(sets),
	)

	buf.WriteString("-- +goose Down\n")
	writeDelete(&buf, "schedules", `"ID"`, collectIDs(sets, func(s mockSet) string { return s.ScheduleID }))
	writeDelete(&buf, "rule_attributes", `"ID"`, collectVariationIDs(sets, func(v ruleVariation) string { return v.RuleAttributeID }))
	writeDelete(&buf, "rules", `"ID"`, collectVariationIDs(sets, func(v ruleVariation) string { return v.RuleID }))
	writeDelete(&buf, "rule_conditions", `"ID"`, collectIDs(sets, func(s mockSet) string { return s.ConditionID }))
	writeDelete(&buf, "decision_rules", `"ID"`, collectIDs(sets, func(s mockSet) string { return s.DecisionRuleID }))
	writeDelete(&buf, "attributes", `"ID"`, collectAttributeIDs(attributes))
	fmt.Fprintf(&buf, "DELETE FROM clen_schema_registry WHERE \"ID\" = %s;\n", sqlStringLiteral(schemaID))
	writeDelete(&buf, "placements", `"ID"`, collectIDs(sets, func(s mockSet) string { return s.PlacementID }))
	writeDelete(&buf, "channels", `"ID"`, collectChannelIDs(channels))

	return buf.String()
}

// ---------------------------------------------------------------------------
// SQL row builders
// ---------------------------------------------------------------------------

func channelValues(channels []channelDef) [][]string {
	out := make([][]string, 0, len(channels))
	for _, ch := range channels {
		out = append(out, []string{
			sqlStringLiteral(ch.ID),
			sqlStringLiteral(ch.ChannelName),
			"NOW()", "NOW()",
		})
	}
	return out
}

func placementValues(sets []mockSet) [][]string {
	out := make([][]string, 0, len(sets))
	for _, s := range sets {
		out = append(out, []string{
			sqlStringLiteral(s.PlacementID),
			sqlStringLiteral(s.PlacementName),
			sqlStringLiteral(s.ChannelID),
			"NOW()", "NOW()",
		})
	}
	return out
}

func attributeValues(schemaID string, attributes []attributeDef) [][]string {
	out := make([][]string, 0, len(attributes))
	for _, a := range attributes {
		out = append(out, []string{
			sqlStringLiteral(a.ID),
			sqlStringLiteral(schemaID),
			sqlStringLiteral(a.FieldName),
			sqlStringLiteral(a.DisplayName),
			sqlStringLiteral(a.DataType),
			a.ValueSQL,
			sqlStringLiteral(a.Description),
			sqlStringLiteral(a.SourceSystem),
			fmt.Sprintf("%t", a.IsActive),
			"NOW()", "NOW()",
		})
	}
	return out
}

func decisionRuleValues(sets []mockSet) [][]string {
	out := make([][]string, 0, len(sets))
	for _, s := range sets {
		out = append(out, []string{
			sqlStringLiteral(s.DecisionRuleID),
			sqlStringLiteral(s.DecisionRuleName),
			sqlStringLiteral("AUDIENCE"),
			sqlStringLiteral("SCORING"),
			sqlStringLiteral(s.ContentPath),
			formatFloat(s.DecisionScore),
			sqlStringLiteral("ACTIVE"),
			"NOW()", "NOW()",
		})
	}
	return out
}

func ruleConditionValues(sets []mockSet) [][]string {
	out := make([][]string, 0, len(sets))
	for _, s := range sets {
		out = append(out, []string{
			sqlStringLiteral(s.ConditionID),
			"1",
			sqlStringLiteral(s.DecisionRuleID),
			sqlStringLiteral(s.AttributeID),
			sqlStringLiteral(s.LogicalOperator),
			sqlStringLiteral(s.ConnectorOperator),
			"NOW()", "NOW()",
		})
	}
	return out
}

func ruleValues(sets []mockSet) [][]string {
	var out [][]string
	for _, s := range sets {
		for _, v := range s.Variations {
			out = append(out, []string{
				sqlStringLiteral(v.RuleID),
				sqlStringLiteral(s.DecisionRuleID),
				sqlStringLiteral(v.VariationName),
				formatFloat(v.RuleScore),
				fmt.Sprintf("%d", v.OrderNo),
				"NOW()", "NOW()",
			})
		}
	}
	return out
}

func ruleAttributeValues(sets []mockSet) [][]string {
	var out [][]string
	for _, s := range sets {
		for _, v := range s.Variations {
			out = append(out, []string{
				sqlStringLiteral(v.RuleAttributeID),
				sqlStringLiteral(v.RuleID),
				sqlStringLiteral(v.AttributeID),
				v.ValueSQL,
				"NOW()", "NOW()",
			})
		}
	}
	return out
}

func scheduleValues(sets []mockSet) [][]string {
	out := make([][]string, 0, len(sets))
	for _, s := range sets {
		out = append(out, []string{
			sqlStringLiteral(s.ScheduleID),
			sqlStringLiteral(s.DecisionRuleID),
			sqlStringLiteral(s.PlacementID),
			sqlStringLiteral("ONCE"),
			s.EffectiveFromExpr,
			s.EffectiveUntilExpr,
			"true",
			"NOW()", "NOW()",
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Redis seed script builder
// ---------------------------------------------------------------------------

func buildRedisSeedScript(attributes []attributeDef, users []userSeed) string {
	attrByField := make(map[string]attributeDef, len(attributes))
	for _, a := range attributes {
		attrByField[a.FieldName] = a
	}

	segAttr := attrByField["segment"]
	ageAttr := attrByField["user_age"]
	regionAttr := attrByField["region"]
	riskAttr := attrByField["risk_level"]

	var buf bytes.Buffer
	buf.WriteString("#!/usr/bin/env sh\n")
	buf.WriteString("# Seed Redis with sample CIS user attributes for local development / testing.\n")
	buf.WriteString("# AUTO-GENERATED by scripts/mockdata — do not edit manually.\n")
	buf.WriteString("# Regenerate: make db-mock-generate-decision-rule\n")
	buf.WriteString("#\n")
	buf.WriteString("# Key format  : cis_id:{cisID}\n")
	buf.WriteString("# Value format: JSON object keyed by attribute UUIDs\n")
	buf.WriteString("#\n")
	buf.WriteString("# Attribute UUIDs (from this generation):\n")
	fmt.Fprintf(&buf, "#   segment    %s  Text   Mass | Affluent | VIP | Young Wealth | SME\n", segAttr.ID)
	fmt.Fprintf(&buf, "#   user_age   %s  Number\n", ageAttr.ID)
	fmt.Fprintf(&buf, "#   region     %s  Text   Bangkok | Central | North | Northeast | South\n", regionAttr.ID)
	fmt.Fprintf(&buf, "#   risk_level %s  Number 1..5\n", riskAttr.ID)
	buf.WriteString("\n")
	buf.WriteString("REDIS_CLI=\"${REDIS_CLI:-redis-cli}\"\n")
	buf.WriteString("\n")
	buf.WriteString("set -e\n")
	buf.WriteString("\n")
	buf.WriteString("seed() {\n")
	buf.WriteString("  CIS_ID=\"$1\"\n")
	buf.WriteString("  PAYLOAD=\"$2\"\n")
	buf.WriteString("  $REDIS_CLI SET \"cis_id:${CIS_ID}\" \"${PAYLOAD}\"\n")
	buf.WriteString("  echo \"  SET cis_id:${CIS_ID}\"\n")
	buf.WriteString("}\n")
	buf.WriteString("\n")
	buf.WriteString("echo \"Seeding Redis user attributes...\"\n")
	buf.WriteString("\n")

	for _, u := range users {
		payload := buildUserPayload(segAttr.ID, ageAttr.ID, regionAttr.ID, riskAttr.ID, u)
		fmt.Fprintf(&buf, "# %s\n", u.Commentary)
		fmt.Fprintf(&buf, "seed %q %q\n", u.CisID, payload)
		buf.WriteString("\n")
	}

	fmt.Fprintf(&buf, "echo \"Done. Seeded %d CIS user attribute records.\"\n", len(users))
	return buf.String()
}

func buildUserPayload(segID, ageID, regionID, riskID string, u userSeed) string {
	m := map[string]any{
		segID:    u.Segment,
		ageID:    u.UserAge,
		regionID: u.Region,
		riskID:   u.RiskLevel,
	}
	b, err := json.Marshal(m)
	if err != nil {
		fail("marshal user payload: %v", err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// SQL write helpers
// ---------------------------------------------------------------------------

func writeInsert(buf *bytes.Buffer, table string, columns []string, values [][]string) {
	fmt.Fprintf(buf, "INSERT INTO %s (%s) VALUES\n", table, strings.Join(columns, ", "))
	for i, row := range values {
		sep := ","
		if i == len(values)-1 {
			sep = ";"
		}
		fmt.Fprintf(buf, "(%s)%s\n", strings.Join(row, ", "), sep)
	}
	buf.WriteString("\n")
}

func writeDelete(buf *bytes.Buffer, table, column string, ids []string) {
	fmt.Fprintf(buf, "DELETE FROM %s WHERE %s IN (\n", table, column)
	for i, id := range ids {
		sep := ","
		if i == len(ids)-1 {
			sep = ""
		}
		fmt.Fprintf(buf, "    %s%s\n", sqlStringLiteral(id), sep)
	}
	buf.WriteString(");\n")
}

// ---------------------------------------------------------------------------
// ID collectors
// ---------------------------------------------------------------------------

func collectChannelIDs(channels []channelDef) []string {
	ids := make([]string, 0, len(channels))
	for _, ch := range channels {
		ids = append(ids, ch.ID)
	}
	return ids
}

func collectAttributeIDs(attributes []attributeDef) []string {
	ids := make([]string, 0, len(attributes))
	for _, a := range attributes {
		ids = append(ids, a.ID)
	}
	return ids
}

func collectIDs(sets []mockSet, getter func(mockSet) string) []string {
	ids := make([]string, 0, len(sets))
	for _, s := range sets {
		ids = append(ids, getter(s))
	}
	return ids
}

func collectVariationIDs(sets []mockSet, getter func(ruleVariation) string) []string {
	var ids []string
	for _, s := range sets {
		for _, v := range s.Variations {
			ids = append(ids, getter(v))
		}
	}
	return ids
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

func sqlStringLiteral(v string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func fail(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
