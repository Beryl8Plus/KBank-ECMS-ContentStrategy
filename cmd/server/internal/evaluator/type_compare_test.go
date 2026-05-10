package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity/enums"
)

// helper: build ParsedUserAttrs + ParsedExpectedValues from one attrID/value pair.
func parsedPair(attrID, actual, expected string) (*ParsedUserAttrs, *ParsedExpectedValues) {
	pa := NewParsedUserAttrs(map[string]json.RawMessage{attrID: json.RawMessage(actual)})
	pe := NewParsedExpectedValues(map[string]json.RawMessage{attrID: json.RawMessage(expected)})
	return pa, pe
}

// ─────────────────────────────────────────────────────────────────────────────
// ParsedUserAttrs Len + nil-receiver
// ─────────────────────────────────────────────────────────────────────────────

func TestParsedUserAttrs_Len(t *testing.T) {
	t.Parallel()
	if got := (&ParsedUserAttrs{}).Len(); got != 0 {
		t.Errorf("empty ParsedUserAttrs Len = %d, want 0", got)
	}
	pa := NewParsedUserAttrs(map[string]json.RawMessage{"x": json.RawMessage(`"y"`)})
	if pa.Len() != 1 {
		t.Errorf("Len = %d", pa.Len())
	}
	var nilPA *ParsedUserAttrs
	if nilPA.Len() != 0 {
		t.Error("nil receiver Len should be 0")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// compareTextParsed — IN, NIN, NEQ, error case
// ─────────────────────────────────────────────────────────────────────────────

func TestCompareTextParsed_AllOps(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()

	for _, c := range []struct {
		name     string
		op       enums.LogicalOperator
		actual   string
		expected string
		want     bool
		wantErr  bool
	}{
		{"EQ-match", enums.LogicalOperatorEQ, `"gold"`, `"gold"`, true, false},
		{"EQ-mismatch", enums.LogicalOperatorEQ, `"gold"`, `"silver"`, false, false},
		{"NEQ-match", enums.LogicalOperatorNEQ, `"a"`, `"b"`, true, false},
		{"NEQ-mismatch", enums.LogicalOperatorNEQ, `"a"`, `"a"`, false, false},
		{"IN-yes", enums.LogicalOperatorIN, `"a"`, `["a","b"]`, true, false},
		{"IN-no", enums.LogicalOperatorIN, `"x"`, `["a","b"]`, false, false},
		{"NIN-yes", enums.LogicalOperatorNIN, `"x"`, `["a","b"]`, true, false},
		{"NIN-no", enums.LogicalOperatorNIN, `"a"`, `["a","b"]`, false, false},
		{"unsupported-op", enums.LogicalOperatorBETWEEN, `"a"`, `"a"`, false, true},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pa, pe := parsedPair(id, c.actual, c.expected)
			actual, _ := pa.GetString(id)
			got, err := compareTextParsed(c.op, actual, id, pe)
			if c.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got=%v err=%v want=%v", got, err, c.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// compareNumberParsed — all ops
// ─────────────────────────────────────────────────────────────────────────────

func TestCompareNumberParsed_AllOps(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()

	for _, c := range []struct {
		name     string
		op       enums.LogicalOperator
		actual   string
		expected string
		want     bool
		wantErr  bool
	}{
		{"EQ", enums.LogicalOperatorEQ, `5`, `5`, true, false},
		{"NEQ", enums.LogicalOperatorNEQ, `5`, `6`, true, false},
		{"LT", enums.LogicalOperatorLT, `5`, `6`, true, false},
		{"LTE", enums.LogicalOperatorLTE, `5`, `5`, true, false},
		{"GT", enums.LogicalOperatorGT, `7`, `5`, true, false},
		{"GTE", enums.LogicalOperatorGTE, `5`, `5`, true, false},
		{"IN-match", enums.LogicalOperatorIN, `2`, `[1,2,3]`, true, false},
		{"IN-miss", enums.LogicalOperatorIN, `9`, `[1,2,3]`, false, false},
		{"BETWEEN-in", enums.LogicalOperatorBETWEEN, `5`, `[1,10]`, true, false},
		{"BETWEEN-out", enums.LogicalOperatorBETWEEN, `15`, `[1,10]`, false, false},
		{"BETWEEN-bad-bounds", enums.LogicalOperatorBETWEEN, `5`, `[1]`, false, true},
		{"unsupported-op", enums.LogicalOperatorNIN, `5`, `5`, false, true},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pa, pe := parsedPair(id, c.actual, c.expected)
			actual, _ := pa.GetNumber(id)
			got, err := compareNumberParsed(c.op, actual, id, pe)
			if c.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got=%v err=%v want=%v", got, err, c.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// compareBooleanParsed — EQ, NEQ, unsupported
// ─────────────────────────────────────────────────────────────────────────────

func TestCompareBooleanParsed(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()
	for _, c := range []struct {
		name    string
		op      enums.LogicalOperator
		actual  string
		expect  string
		want    bool
		wantErr bool
	}{
		{"EQ-match", enums.LogicalOperatorEQ, `true`, `true`, true, false},
		{"EQ-mismatch", enums.LogicalOperatorEQ, `true`, `false`, false, false},
		{"NEQ-match", enums.LogicalOperatorNEQ, `true`, `false`, true, false},
		{"unsupported", enums.LogicalOperatorLT, `true`, `true`, false, true},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pa, pe := parsedPair(id, c.actual, c.expect)
			actual, _ := pa.GetBool(id)
			got, err := compareBooleanParsed(c.op, actual, id, pe)
			if c.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got=%v err=%v want=%v", got, err, c.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// compareDateParsed — EQ/NEQ/LT/LTE/GT/GTE/BETWEEN/unsupported
// ─────────────────────────────────────────────────────────────────────────────

func TestCompareDateParsed(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()
	for _, c := range []struct {
		name    string
		op      enums.LogicalOperator
		actual  string
		expect  string
		want    bool
		wantErr bool
	}{
		{"EQ", enums.LogicalOperatorEQ, `"2026-01-01"`, `"2026-01-01"`, true, false},
		{"NEQ", enums.LogicalOperatorNEQ, `"2026-01-01"`, `"2026-01-02"`, true, false},
		{"LT", enums.LogicalOperatorLT, `"2025-12-31"`, `"2026-01-01"`, true, false},
		{"LTE-eq", enums.LogicalOperatorLTE, `"2026-01-01"`, `"2026-01-01"`, true, false},
		{"GT", enums.LogicalOperatorGT, `"2026-02-01"`, `"2026-01-01"`, true, false},
		{"GTE-eq", enums.LogicalOperatorGTE, `"2026-01-01"`, `"2026-01-01"`, true, false},
		{"BETWEEN-in", enums.LogicalOperatorBETWEEN, `"2026-01-15"`, `["2026-01-01","2026-01-31"]`, true, false},
		{"BETWEEN-out", enums.LogicalOperatorBETWEEN, `"2026-02-15"`, `["2026-01-01","2026-01-31"]`, false, false},
		{"unsupported", enums.LogicalOperatorIN, `"2026-01-01"`, `"2026-01-01"`, false, true},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pa, pe := parsedPair(id, c.actual, c.expect)
			actual, _ := pa.GetDate(id)
			got, err := compareDateParsed(c.op, actual, id, pe)
			if c.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got=%v err=%v want=%v", got, err, c.want)
			}
		})
	}
}
