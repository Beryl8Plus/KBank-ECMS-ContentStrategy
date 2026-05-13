package evaluator

import (
	"encoding/json"
	"strconv"
	"strings"

	"kbank-ecms/internal/domain/entity/enums"
)

// sentinelKind classifies an expected value's role in evaluation.
type sentinelKind int

const (
	sentinelNone sentinelKind = iota
	sentinelAny               // "ANY" / "any" — always matches
	sentinelNull              // "NULL" / "null" — matches when user value is null
	sentinelList              // "v1^v2^..." — caret-separated list
)

// extractSentinel inspects a raw JSON expected value and classifies it.
// It returns sentinelNone when raw does not decode to a string, when the
// string is empty after trimming, or when the trimmed string is not a
// recognised sentinel.
//
// For sentinelList the returned payload is the trimmed string (still
// containing the "^" separators). For sentinelAny / sentinelNull the
// payload is the empty string.
func extractSentinel(raw json.RawMessage) (sentinelKind, string) {
	if len(raw) == 0 {
		return sentinelNone, ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return sentinelNone, ""
	}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return sentinelNone, ""
	}
	if strings.EqualFold(trimmed, "ANY") {
		return sentinelAny, ""
	}
	if strings.EqualFold(trimmed, "NULL") {
		return sentinelNull, ""
	}
	if strings.Contains(trimmed, "^") {
		return sentinelList, trimmed
	}
	return sentinelNone, ""
}

// applySentinel inspects rawExpected and, if it is a sentinel form, produces
// the comparison result directly. Returns handled=false when no sentinel
// applies (the caller must fall through to the normal comparator).
//
// Only Text and Number data types participate in sentinel handling. Date and
// Boolean attributes always return (false, false, nil).
func applySentinel(
	dt enums.AttributeDataType,
	op enums.LogicalOperator,
	rawExpected json.RawMessage,
	parsed *ParsedUserAttrs,
	attrKey string,
) (bool, bool, error) {
	if dt != enums.AttributeDataTypeText && dt != enums.AttributeDataTypeNumber {
		return false, false, nil
	}

	kind, payload := extractSentinel(rawExpected)
	switch kind {
	case sentinelAny:
		return true, true, nil
	case sentinelNull:
		switch op {
		case enums.LogicalOperatorEQ:
			return parsed.IsNull(attrKey), true, nil
		case enums.LogicalOperatorNEQ:
			return !parsed.IsNull(attrKey), true, nil
		default:
			// NULL is only meaningful with = / !=. Anything else: handled, non-match.
			return false, true, nil
		}
	case sentinelList:
		tokens := splitCaretTokens(payload)
		if len(tokens) == 0 {
			return false, true, nil
		}
		switch dt {
		case enums.AttributeDataTypeText:
			return matchTextList(op, tokens, parsed, attrKey)
		case enums.AttributeDataTypeNumber:
			nums, ok := parseCaretNumbers(tokens)
			if !ok {
				return false, true, nil
			}
			return matchNumberList(op, nums, parsed, attrKey)
		}
		return false, true, nil
	}
	return false, false, nil
}

// splitCaretTokens splits a caret-list payload on "^", trims each token, and
// drops empty tokens. Returns nil when no non-empty tokens remain.
func splitCaretTokens(payload string) []string {
	parts := strings.Split(payload, "^")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseCaretNumbers parses each token in tokens as float64. Returns (nil,
// false) when tokens is empty or any token fails to parse. The all-or-nothing
// semantic ensures a malformed token cannot silently shrink the IN/NIN set.
func parseCaretNumbers(tokens []string) ([]float64, bool) {
	if len(tokens) == 0 {
		return nil, false
	}
	out := make([]float64, 0, len(tokens))
	for _, t := range tokens {
		n, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

// matchTextList implements caret-list comparison for Text attributes.
// EQ promotes to IN, NEQ promotes to NOT IN. Other operators return
// (false, true, nil) — handled non-match.
func matchTextList(op enums.LogicalOperator, tokens []string, parsed *ParsedUserAttrs, attrKey string) (bool, bool, error) {
	actual, ok := parsed.GetString(attrKey)
	if !ok {
		return false, true, nil
	}
	contains := func() bool {
		for _, t := range tokens {
			if strings.EqualFold(t, actual) {
				return true
			}
		}
		return false
	}
	switch op {
	case enums.LogicalOperatorEQ, enums.LogicalOperatorIN:
		return contains(), true, nil
	case enums.LogicalOperatorNEQ, enums.LogicalOperatorNIN:
		return !contains(), true, nil
	default:
		return false, true, nil
	}
}

// matchNumberList implements caret-list comparison for Number attributes.
// EQ promotes to IN, NEQ promotes to NOT IN. Other operators return
// (false, true, nil) — handled non-match.
func matchNumberList(op enums.LogicalOperator, nums []float64, parsed *ParsedUserAttrs, attrKey string) (bool, bool, error) {
	actual, ok := parsed.GetNumber(attrKey)
	if !ok {
		return false, true, nil
	}
	contains := func() bool {
		for _, n := range nums {
			if actual == n {
				return true
			}
		}
		return false
	}
	switch op {
	case enums.LogicalOperatorEQ, enums.LogicalOperatorIN:
		return contains(), true, nil
	case enums.LogicalOperatorNEQ, enums.LogicalOperatorNIN:
		return !contains(), true, nil
	default:
		return false, true, nil
	}
}
