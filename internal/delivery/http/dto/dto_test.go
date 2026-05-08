package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"kbank-ecms/internal/domain/entity/enums"
)

func TestCustomerRequest_IsEmpty(t *testing.T) {
	t.Parallel()
	if !(CustomerRequest{}).IsEmpty() {
		t.Error("zero value should be empty")
	}
	if (CustomerRequest{Type: CustomerIdTypeCISID, CIS_ID: "1"}).IsEmpty() {
		t.Error("populated CIS_ID should not be empty")
	}
}

func TestCustomerRequest_TypeName(t *testing.T) {
	t.Parallel()
	r := CustomerRequest{Type: CustomerIdTypeCISID}
	if r.TypeName() != string(CustomerIdTypeCISID) {
		t.Errorf("TypeName mismatch: %q", r.TypeName())
	}
}

func TestCustomerRequest_Value(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		req  CustomerRequest
		want string
	}{
		{"cis", CustomerRequest{Type: CustomerIdTypeCISID, CIS_ID: "C1"}, "C1"},
		{"ip", CustomerRequest{Type: CustomerIdTypeIPID, IP_ID: "I1"}, "I1"},
		{"kplus", CustomerRequest{Type: CustomerIdTypeKPlusMobileNumber, KPlusMobileNumber: "K1"}, "K1"},
		{"line", CustomerRequest{Type: CustomerIdTypeLineUUID, LineUUID: "L1"}, "L1"},
		{"unknown", CustomerRequest{Type: "OTHER"}, ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.req.Value(); got != c.want {
				t.Errorf("Value() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestCacheStatusResponse_MarshalJSON_RoundsPct(t *testing.T) {
	t.Parallel()
	r := CacheStatusResponse{IsMemPressure: true, MemoryUsagePct: 0.123456789, CacheKeys: []string{"k1"}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"memoryUsagePct":0.1235`) {
		t.Errorf("expected rounded pct, got %s", data)
	}
}

func TestContentResult_ToResponse_StripsLogicFields(t *testing.T) {
	t.Parallel()
	src := ContentResult{
		LogicHash:      "h",
		LogicExpr:      "x",
		LogicEval:      true,
		EvaluatedAt:    "2026-01-01T00:00:00Z",
		DecisionRuleId: "r1",
		RuleSetType:    "MASS",
		ContentPath:    "/x",
		Source:         "src",
		Score:          1.5,
		StartDateTime:  "s",
		EndDateTime:    "e",
		Conditions:     []LogicCondition{{ConditionID: "c1"}},
	}
	got := src.ToResponse()
	if got.LogicHash != "" || got.LogicExpr != "" || got.LogicEval {
		t.Error("logic fields should be stripped")
	}
	if got.Conditions != nil {
		t.Error("Conditions should be stripped")
	}
	if got.DecisionRuleId != "r1" || got.Score != 1.5 {
		t.Error("public fields should round-trip")
	}
}

func TestToContentResultResponses(t *testing.T) {
	t.Parallel()
	in := []ContentResult{
		{DecisionRuleId: "a", LogicHash: "x"},
		{DecisionRuleId: "b", LogicHash: "y"},
	}
	out := ToContentResultResponses(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	for _, r := range out {
		if r.LogicHash != "" {
			t.Error("logic fields should be stripped")
		}
	}
}

func TestAPIResponse_ToSuccess_OnlyWhenEmpty(t *testing.T) {
	t.Parallel()
	r := &APIResponse{}
	r.ToSuccess()
	if r.Code != enums.SuccessResponse.String() {
		t.Errorf("expected success code, got %q", r.Code)
	}
	r2 := &APIResponse{Code: "EXISTING"}
	r2.ToSuccess()
	if r2.Code != "EXISTING" {
		t.Error("ToSuccess should not overwrite existing code")
	}
	r3 := &APIResponse{Error: "boom"}
	r3.ToSuccess()
	if r3.Code != "" {
		t.Error("ToSuccess should not set code when Error is set")
	}
}

func TestAPIResponse_MarshalJSON_AutoSuccess(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(APIResponse{Data: map[string]int{"n": 1}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"code":"`+enums.SuccessResponse.String()+`"`) {
		t.Errorf("expected auto-success code in JSON, got %s", data)
	}
}
