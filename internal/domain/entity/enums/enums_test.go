package enums

import "testing"

// Each enum type follows the same shape (String / IsValid / Parse) so the
// tests below are mechanical: round-trip the canonical constants, reject
// a known-invalid value, and (where applicable) exercise the type's helper
// methods. Coverage stays high because each method has zero-control-flow
// or a single switch.

func TestAttributeDataType(t *testing.T) {
	t.Parallel()
	for _, v := range []AttributeDataType{
		AttributeDataTypeText, AttributeDataTypeDate,
		AttributeDataTypeNumber, AttributeDataTypeBoolean,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if AttributeDataType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := AttributeDataType("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestCalendarType(t *testing.T) {
	t.Parallel()
	for _, v := range []CalendarType{
		CalendarTypeHoliday, CalendarTypePersonal, CalendarTypeCustom,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, _ := v.Parse(string(v)); got != v {
			t.Errorf("Parse(%q) = %q", v, got)
		}
	}
	if CalendarType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := CalendarType("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestConditionType(t *testing.T) {
	t.Parallel()
	if !ConditionTypeCondition.IsValid() || !ConditionTypeGroup.IsValid() {
		t.Error("canonical values must be valid")
	}
	if ConditionType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
}

func TestConnectorOperator(t *testing.T) {
	t.Parallel()
	for _, v := range []ConnectorOperator{ConnectorOperatorAND, ConnectorOperatorOR} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if ConnectorOperator("XOR").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := ConnectorOperator("").Parse("XOR"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestRequestType(t *testing.T) {
	t.Parallel()
	for _, v := range []RequestType{
		RequestTypePersonalizedContent, RequestTypeStaticContent, RequestTypeArticleCategory,
	} {
		if !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
	}
	if RequestType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
}

func TestDecisionRuleStatus(t *testing.T) {
	t.Parallel()
	for _, v := range []DecisionRuleStatus{
		DecisionRuleStatusDraft, DecisionRuleStatusActive, DecisionRuleStatusInactive,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if DecisionRuleStatus("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := DecisionRuleStatus("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestDecisionRuleSubStatus(t *testing.T) {
	t.Parallel()
	for _, v := range []DecisionRuleSubStatus{
		DecisionRuleSubStatusNA, DecisionRuleSubStatusMissing,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if DecisionRuleSubStatus("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := DecisionRuleSubStatus("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestEvaluateType(t *testing.T) {
	t.Parallel()
	for _, v := range []EvaluateType{
		EvaluateTypeScoring, EvaluateTypeSegment, EvaluateTypeEligible,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if EvaluateType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := EvaluateType("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestDecisionType(t *testing.T) {
	t.Parallel()
	for _, v := range []DecisionType{
		DecisionTypeMass, DecisionTypeAudience, DecisionTypeSalesTarget, DecisionTypeNonSales,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	// Values returns the full set
	if got := DecisionTypeMass.Values(); len(got) != 4 {
		t.Errorf("Values() returned %d items, want 4", len(got))
	}
	// IsCampaign — only AUDIENCE and SALES_TARGET return true
	if !DecisionTypeAudience.IsCampaign() || !DecisionTypeSalesTarget.IsCampaign() {
		t.Error("AUDIENCE and SALES_TARGET should be campaigns")
	}
	if DecisionTypeMass.IsCampaign() || DecisionTypeNonSales.IsCampaign() {
		t.Error("MASS and NON_SALES should not be campaigns")
	}
	if DecisionType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := DecisionType("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestFeature(t *testing.T) {
	t.Parallel()
	if !FeatureContentDecisionRule.IsValid() {
		t.Error("canonical value should be valid")
	}
	if FeatureContentDecisionRule.String() != string(FeatureContentDecisionRule) {
		t.Error("String mismatch")
	}
	if got, err := FeatureContentDecisionRule.Parse(string(FeatureContentDecisionRule)); err != nil || got != FeatureContentDecisionRule {
		t.Errorf("Parse failed: %v, %v", got, err)
	}
	if vals := FeatureContentDecisionRule.Values(); len(vals) == 0 {
		t.Error("Values() should not be empty")
	}
	if Feature("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := Feature("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestLogicalOperator(t *testing.T) {
	t.Parallel()
	for _, v := range []LogicalOperator{
		LogicalOperatorLT, LogicalOperatorLTE,
		LogicalOperatorGT, LogicalOperatorGTE,
		LogicalOperatorEQ, LogicalOperatorNEQ,
		LogicalOperatorIN, LogicalOperatorNIN, LogicalOperatorBETWEEN,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if LogicalOperator("??").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := LogicalOperator("").Parse("??"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestOccurrenceSource(t *testing.T) {
	t.Parallel()
	for _, v := range []OccurrenceSource{
		OccurrenceSourceRecurrence, OccurrenceSourceCalendar, OccurrenceSourceManual,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if OccurrenceSource("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := OccurrenceSource("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestOccurrenceStatus(t *testing.T) {
	t.Parallel()
	for _, v := range []OccurrenceStatus{
		OccurrenceStatusActive, OccurrenceStatusCancelled, OccurrenceStatusExpired,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if OccurrenceStatus("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := OccurrenceStatus("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestRecurrenceType(t *testing.T) {
	t.Parallel()
	for _, v := range []RecurrenceType{
		RecurrenceTypeOnce, RecurrenceTypeRRule, RecurrenceTypeCalendar,
	} {
		if v.String() != string(v) || !v.IsValid() {
			t.Errorf("%v should be valid", v)
		}
		if got, err := v.Parse(string(v)); err != nil || got != v {
			t.Errorf("Parse(%q) = (%v, %v)", v, got, err)
		}
	}
	if RecurrenceType("nope").IsValid() {
		t.Error("unknown value should be invalid")
	}
	if _, err := RecurrenceType("").Parse("nope"); err == nil {
		t.Error("Parse should reject invalid value")
	}
}

func TestResponseCode(t *testing.T) {
	t.Parallel()
	for _, v := range []ResponseCode{
		SuccessResponse, ErrorCodeBadRequest, ErrorCodeInternalError,
	} {
		if v.String() != string(v) {
			t.Errorf("String mismatch for %v", v)
		}
	}
}
