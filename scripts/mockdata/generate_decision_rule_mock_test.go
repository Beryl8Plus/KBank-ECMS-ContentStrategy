package main

import (
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

// helper to seed deterministically before each generation step
func seed(t *testing.T) {
	t.Helper()
	if err := gofakeit.Seed(20260101); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestBuildChannels(t *testing.T) {
	seed(t)
	got := buildChannels()
	if len(got) != 4 {
		t.Errorf("expected 4 channels, got %d", len(got))
	}
	for _, c := range got {
		if c.ID == "" || c.ChannelName == "" {
			t.Errorf("channel missing fields: %+v", c)
		}
	}
}

func TestBuildPlacements_OneToOneWithChannels(t *testing.T) {
	seed(t)
	channels := buildChannels()
	got := buildPlacements(channels)
	if len(got) != len(channels) {
		t.Errorf("placements/channels mismatch: %d vs %d", len(got), len(channels))
	}
	for i, p := range got {
		if p.ChannelID != channels[i].ID {
			t.Errorf("placement[%d].ChannelID = %q, want %q", i, p.ChannelID, channels[i].ID)
		}
	}
}

func TestBuildAttributes_ExpectedFields(t *testing.T) {
	seed(t)
	got := buildAttributes()
	if len(got) < 4 {
		t.Errorf("expected ≥ 4 attributes, got %d", len(got))
	}
	want := map[string]bool{"segment": false, "user_age": false, "region": false, "risk_level": false}
	for _, a := range got {
		if _, ok := want[a.FieldName]; ok {
			want[a.FieldName] = true
		}
		if a.ID == "" || a.DataType == "" {
			t.Errorf("attribute missing fields: %+v", a)
		}
	}
	for k, found := range want {
		if !found {
			t.Errorf("missing canonical attribute %q", k)
		}
	}
}

func TestBuildMockSets(t *testing.T) {
	seed(t)
	channels := buildChannels()
	placements := buildPlacements(channels)
	attributes := buildAttributes()

	got := buildMockSets(5, placements, attributes)
	if len(got) != 5 {
		t.Errorf("expected 5 sets, got %d", len(got))
	}
	for _, set := range got {
		if set.DecisionRuleID == "" || set.PlacementID == "" || len(set.Variations) == 0 {
			t.Errorf("set missing fields: %+v", set)
		}
		if set.ScheduleID == "" {
			t.Errorf("schedule missing")
		}
	}
}

func TestBuildUsers(t *testing.T) {
	seed(t)
	attrs := buildAttributes()
	users := buildUsers(attrs)
	if len(users) == 0 {
		t.Error("expected non-empty users slice")
	}
	for _, u := range users {
		if u.CisID == "" || u.Segment == "" {
			t.Errorf("user missing fields: %+v", u)
		}
	}
}

func TestBuildCondition_Coverage(t *testing.T) {
	seed(t)
	segments := []string{"Mass", "Affluent"}
	regions := []string{"Bangkok", "Central"}
	for _, field := range []string{"segment", "user_age", "region", "risk_level", "unknown"} {
		op, val, label := buildCondition(field, segments, regions)
		if op == "" || val == "" || label == "" {
			t.Errorf("buildCondition(%q): empty values: op=%q val=%q label=%q", field, op, val, label)
		}
	}
}

func TestBuildSQL_HasExpectedTableInserts(t *testing.T) {
	seed(t)
	channels := buildChannels()
	placements := buildPlacements(channels)
	attributes := buildAttributes()
	sets := buildMockSets(3, placements, attributes)

	sql := buildSQL(20260101, "schema-id", channels, placements, attributes, sets)
	for _, want := range []string{
		"INSERT INTO channels",
		"INSERT INTO placements",
		"INSERT INTO attributes",
		"INSERT INTO decision_rules",
		"INSERT INTO rules",
		"INSERT INTO rule_attributes",
		"INSERT INTO rule_conditions",
		"INSERT INTO schedules",
		"DELETE FROM",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q", want)
		}
	}
}

func TestBuildRedisSeedScript_HasSeedCommands(t *testing.T) {
	seed(t)
	attrs := buildAttributes()
	users := buildUsers(attrs)
	script := buildRedisSeedScript(attrs, users)
	if !strings.Contains(script, "REDIS_CLI") {
		t.Error("script missing REDIS_CLI variable")
	}
	if !strings.Contains(script, "seed ") {
		t.Error("script missing seed function calls")
	}
}

func TestSqlStringLiteral(t *testing.T) {
	if got := sqlStringLiteral("O'Brien"); got != "'O''Brien'" {
		t.Errorf("escaping failed: %q", got)
	}
	if got := sqlStringLiteral("plain"); got != "'plain'" {
		t.Errorf("plain quoting: %q", got)
	}
}

func TestFormatFloat(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{1.5, "1.5"},
		{2.0, "2.0"},
		{0.123, "0.1"}, // %.1f truncates
	} {
		if got := formatFloat(c.in); got != c.want {
			t.Errorf("formatFloat(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRound1(t *testing.T) {
	if got := round1(1.234); got != 1.2 {
		t.Errorf("round1(1.234) = %v", got)
	}
	if got := round1(1.25); got != 1.3 {
		t.Errorf("round1(1.25) = %v", got)
	}
}

func TestFindLatestMockFile_NoMatch(t *testing.T) {
	got, ok := findLatestMockFile("non-existent-mock-name-zzz")
	if ok || got != "" {
		t.Errorf("expected (\"\", false) for non-existent name, got (%q, %v)", got, ok)
	}
}
