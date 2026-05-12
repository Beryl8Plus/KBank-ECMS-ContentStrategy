package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"kbank-ecms/internal/domain/entity/enums"
)

func TestExtractSentinel(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantKind    sentinelKind
		wantPayload string
	}{
		{"AnyUpper", `"ANY"`, sentinelAny, ""},
		{"AnyLower", `"any"`, sentinelAny, ""},
		{"AnyMixedCase", `"AnY"`, sentinelAny, ""},
		{"AnyTrimmed", `"  ANY  "`, sentinelAny, ""},
		{"NullUpper", `"NULL"`, sentinelNull, ""},
		{"NullLower", `"null"`, sentinelNull, ""},
		{"JsonNullLiteral_NotSentinel", `null`, sentinelNone, ""},
		{"List_TwoTokens", `"a^b"`, sentinelList, "a^b"},
		{"List_ThreeTokens", `"v1^v2^v3"`, sentinelList, "v1^v2^v3"},
		{"List_Trimmed", `"  a^b  "`, sentinelList, "a^b"},
		{"PlainString_NotSentinel", `"gold"`, sentinelNone, ""},
		{"Number_NotSentinel", `42`, sentinelNone, ""},
		{"Array_NotSentinel", `["a","b"]`, sentinelNone, ""},
		{"Object_NotSentinel", `{"k":"v"}`, sentinelNone, ""},
		{"Empty_NotSentinel", ``, sentinelNone, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, payload := extractSentinel(json.RawMessage(tc.raw))
			assert.Equal(t, tc.wantKind, kind)
			assert.Equal(t, tc.wantPayload, payload)
		})
	}
}

func TestSplitCaretTokens(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"TwoTokens", "a^b", []string{"a", "b"}},
		{"ThreeTokens", "v1^v2^v3", []string{"v1", "v2", "v3"}},
		{"TrimsSpaces", " a ^ b ^ c ", []string{"a", "b", "c"}},
		{"DropsEmptyTokens", "a^^b", []string{"a", "b"}},
		{"DropsWhitespaceOnlyTokens", "a^   ^b", []string{"a", "b"}},
		{"SingleTokenNoCaret", "a", []string{"a"}},
		{"AllEmpty_ReturnsNil", "^^^", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCaretTokens(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseCaretNumbers(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		want   []float64
		wantOK bool
	}{
		{"AllIntegers", []string{"1", "2", "3"}, []float64{1, 2, 3}, true},
		{"Floats", []string{"1.5", "2.5"}, []float64{1.5, 2.5}, true},
		{"NegativeAndZero", []string{"-1", "0", "1"}, []float64{-1, 0, 1}, true},
		{"NonNumericToken_Rejected", []string{"1", "x", "3"}, nil, false},
		{"EmptyInput_Rejected", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseCaretNumbers(tc.input)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestApplySentinel_Any(t *testing.T) {
	user := NewParsedUserAttrs(map[string]json.RawMessage{
		"k": json.RawMessage(`"gold"`),
	})

	cases := []struct {
		name string
		dt   enums.AttributeDataType
		op   enums.LogicalOperator
	}{
		{"Text_EQ", enums.AttributeDataTypeText, enums.LogicalOperatorEQ},
		{"Text_NEQ", enums.AttributeDataTypeText, enums.LogicalOperatorNEQ},
		{"Text_IN", enums.AttributeDataTypeText, enums.LogicalOperatorIN},
		{"Number_EQ", enums.AttributeDataTypeNumber, enums.LogicalOperatorEQ},
		{"Number_GT", enums.AttributeDataTypeNumber, enums.LogicalOperatorGT},
	}
	for _, tc := range cases {
		t.Run(tc.name+"_HandledAndMatches", func(t *testing.T) {
			res, handled, err := applySentinel(tc.dt, tc.op, json.RawMessage(`"ANY"`), user, "k")
			assert.NoError(t, err)
			assert.True(t, handled)
			assert.True(t, res)
		})
	}

	t.Run("Date_ANY_NotHandled", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeDate, enums.LogicalOperatorEQ, json.RawMessage(`"ANY"`), user, "k")
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.False(t, res)
	})

	t.Run("Boolean_ANY_NotHandled", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeBoolean, enums.LogicalOperatorEQ, json.RawMessage(`"ANY"`), user, "k")
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.False(t, res)
	})

	t.Run("PlainString_NotHandled", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"gold"`), user, "k")
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.False(t, res)
	})
}

func TestApplySentinel_Null(t *testing.T) {
	withVal := NewParsedUserAttrs(map[string]json.RawMessage{
		"k": json.RawMessage(`"gold"`),
	})
	withJSONNull := NewParsedUserAttrs(map[string]json.RawMessage{
		"k": json.RawMessage(`null`),
	})
	withAbsent := NewParsedUserAttrs(map[string]json.RawMessage{
		"other": json.RawMessage(`"x"`),
	})

	t.Run("EQ_UserNull_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"NULL"`), withJSONNull, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("EQ_UserAbsent_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"NULL"`), withAbsent, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("EQ_UserNonNull_DoesNotMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"NULL"`), withVal, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("NEQ_UserNonNull_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorNEQ, json.RawMessage(`"NULL"`), withVal, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("NEQ_UserNull_DoesNotMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorNEQ, json.RawMessage(`"NULL"`), withJSONNull, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("Number_EQ_AbsentMatches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorEQ, json.RawMessage(`"null"`), withAbsent, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("GT_AlwaysHandledNonMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorGT, json.RawMessage(`"NULL"`), withVal, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("IN_AlwaysHandledNonMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorIN, json.RawMessage(`"NULL"`), withVal, "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
}

func TestApplySentinel_List_Text(t *testing.T) {
	user := func(val string) *ParsedUserAttrs {
		return NewParsedUserAttrs(map[string]json.RawMessage{"k": json.RawMessage(val)})
	}

	t.Run("EQ_PromotesToIN_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"gold^silver"`), user(`"gold"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("EQ_PromotesToIN_NotMember", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorEQ, json.RawMessage(`"gold^silver"`), user(`"bronze"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("IN_DirectMatch_CaseInsensitive", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorIN, json.RawMessage(`"GOLD^silver"`), user(`"gold"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("NEQ_PromotesToNIN_NotMember_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorNEQ, json.RawMessage(`"gold^silver"`), user(`"bronze"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("NIN_DirectMatch_DoesNotMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorNIN, json.RawMessage(`"gold^silver"`), user(`"gold"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("GT_HandledNonMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeText, enums.LogicalOperatorGT, json.RawMessage(`"a^b"`), user(`"a"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
}

func TestApplySentinel_List_Number(t *testing.T) {
	user := func(val string) *ParsedUserAttrs {
		return NewParsedUserAttrs(map[string]json.RawMessage{"k": json.RawMessage(val)})
	}

	t.Run("EQ_Member_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorEQ, json.RawMessage(`"1^2^3"`), user(`2`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
	t.Run("IN_NotMember", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorIN, json.RawMessage(`"1^2^3"`), user(`5`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("MalformedToken_HandledNonMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorIN, json.RawMessage(`"1^x^3"`), user(`1`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("UserNonNumeric_HandledNonMatch", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorIN, json.RawMessage(`"1^2"`), user(`"notanumber"`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.False(t, res)
	})
	t.Run("NIN_NotMember_Matches", func(t *testing.T) {
		res, handled, err := applySentinel(enums.AttributeDataTypeNumber, enums.LogicalOperatorNIN, json.RawMessage(`"1^2"`), user(`5`), "k")
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.True(t, res)
	})
}
