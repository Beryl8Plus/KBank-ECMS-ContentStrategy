package evaluator

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// ---------------------------------------------------------------------------
// parsedEntry — shared lazy-parse cache for a single JSON attribute value.
//
// Both ParsedUserAttrs (actual values) and ParsedExpectedValues (expected values)
// use this struct. Each type caches independently; fields unused by one type
// are simply left at their zero values.
//
// Each concrete type has an independent "attempted" flag so a failed parse is
// never retried, regardless of which type is requested first.
// ---------------------------------------------------------------------------

type parsedEntry struct {
	raw                 json.RawMessage
	str                 string
	strOK               bool
	strAttempted        bool
	strs                []string
	strsOK              bool
	strsAttempted       bool
	num                 float64
	numOK               bool
	numAttempted        bool
	nums                []float64
	numsOK              bool
	numsAttempted       bool
	boolVal             bool
	boolOK              bool
	boolAttempted       bool
	date                time.Time
	dateOK              bool
	dateAttempted       bool
	dateBounds          [2]time.Time
	dateBoundsOK        bool
	dateBoundsAttempted bool
}

// ---------------------------------------------------------------------------
// Shared getter helpers — operate on *parsedEntry, nil-safe.
// Both ParsedUserAttrs and ParsedExpectedValues delegate to these.
// ---------------------------------------------------------------------------

func getStr(pv *parsedEntry) (string, bool) {
	if pv == nil {
		return "", false
	}
	if !pv.strAttempted {
		pv.strAttempted = true
		if err := json.Unmarshal(pv.raw, &pv.str); err == nil {
			pv.strOK = true
		}
	}
	return pv.str, pv.strOK
}

func getStrSlice(pv *parsedEntry) ([]string, bool) {
	if pv == nil {
		return nil, false
	}
	if !pv.strsAttempted {
		pv.strsAttempted = true
		if err := json.Unmarshal(pv.raw, &pv.strs); err == nil {
			pv.strsOK = true
		}
	}
	return pv.strs, pv.strsOK
}

func getNum(pv *parsedEntry) (float64, bool) {
	if pv == nil {
		return 0, false
	}
	if !pv.numAttempted {
		pv.numAttempted = true
		if err := json.Unmarshal(pv.raw, &pv.num); err == nil {
			pv.numOK = true
		}
	}
	return pv.num, pv.numOK
}

func getNumSlice(pv *parsedEntry) ([]float64, bool) {
	if pv == nil {
		return nil, false
	}
	if !pv.numsAttempted {
		pv.numsAttempted = true
		if err := json.Unmarshal(pv.raw, &pv.nums); err == nil {
			pv.numsOK = true
		}
	}
	return pv.nums, pv.numsOK
}

func getBool(pv *parsedEntry) (bool, bool) {
	if pv == nil {
		return false, false
	}
	if !pv.boolAttempted {
		pv.boolAttempted = true
		if err := json.Unmarshal(pv.raw, &pv.boolVal); err == nil {
			pv.boolOK = true
		}
	}
	return pv.boolVal, pv.boolOK
}

func getDate(pv *parsedEntry) (time.Time, bool) {
	if pv == nil {
		return time.Time{}, false
	}
	if !pv.dateAttempted {
		pv.dateAttempted = true
		var s string
		if err := json.Unmarshal(pv.raw, &s); err == nil {
			if t, err := parseDate(s); err == nil {
				pv.date = t
				pv.dateOK = true
			}
		}
	}
	return pv.date, pv.dateOK
}

func getDateBounds(pv *parsedEntry) ([2]time.Time, bool) {
	if pv == nil {
		return [2]time.Time{}, false
	}
	if !pv.dateBoundsAttempted {
		pv.dateBoundsAttempted = true
		var bounds []string
		if err := json.Unmarshal(pv.raw, &bounds); err == nil && len(bounds) == 2 {
			lo, err1 := parseDate(bounds[0])
			hi, err2 := parseDate(bounds[1])
			if err1 == nil && err2 == nil {
				pv.dateBounds = [2]time.Time{lo, hi}
				pv.dateBoundsOK = true
			}
		}
	}
	return pv.dateBounds, pv.dateBoundsOK
}

// ---------------------------------------------------------------------------
// ParsedUserAttrs — pre-parsed actual-value cache (user attributes)
//
// Not concurrency-safe; intended for single-goroutine use per request
// (matches ParsedExpectedValues).
// ---------------------------------------------------------------------------

type ParsedUserAttrs struct {
	raw   map[string]json.RawMessage
	cache map[string]*parsedEntry
}

func NewParsedUserAttrs(attrs map[string]json.RawMessage) *ParsedUserAttrs {
	if attrs == nil {
		return nil
	}
	return &ParsedUserAttrs{raw: attrs, cache: make(map[string]*parsedEntry, len(attrs))}
}

func (p *ParsedUserAttrs) Raw(attrID string) (json.RawMessage, bool) {
	if p == nil {
		return nil, false
	}
	v, ok := p.raw[attrID]
	return v, ok
}

func (p *ParsedUserAttrs) Len() int {
	if p == nil {
		return 0
	}
	return len(p.raw)
}

func (p *ParsedUserAttrs) get(attrID string) *parsedEntry {
	if p == nil {
		return nil
	}
	pv, ok := p.cache[attrID]
	if !ok {
		raw, exists := p.raw[attrID]
		if !exists {
			return nil
		}
		pv = &parsedEntry{raw: raw}
		p.cache[attrID] = pv
	}
	return pv
}

func (p *ParsedUserAttrs) GetString(attrID string) (string, bool)  { return getStr(p.get(attrID)) }
func (p *ParsedUserAttrs) GetNumber(attrID string) (float64, bool) { return getNum(p.get(attrID)) }
func (p *ParsedUserAttrs) GetBool(attrID string) (bool, bool)      { return getBool(p.get(attrID)) }
func (p *ParsedUserAttrs) GetDate(attrID string) (time.Time, bool) { return getDate(p.get(attrID)) }

// ---------------------------------------------------------------------------
// ParsedExpectedValues — pre-parsed expected-value cache (rule attributes)
//
// Not concurrency-safe; intended for single-goroutine use per request.
// ---------------------------------------------------------------------------

type ParsedExpectedValues struct {
	raw   map[string]json.RawMessage
	cache map[string]*parsedEntry
}

func NewParsedExpectedValues(raw map[string]json.RawMessage) *ParsedExpectedValues {
	if raw == nil {
		return nil
	}
	return &ParsedExpectedValues{raw: raw, cache: make(map[string]*parsedEntry, len(raw))}
}

func (p *ParsedExpectedValues) Has(attrID string) bool {
	if p == nil {
		return false
	}
	_, ok := p.raw[attrID]
	return ok
}

func (p *ParsedExpectedValues) get(attrID string) *parsedEntry {
	if p == nil {
		return nil
	}
	pv, ok := p.cache[attrID]
	if !ok {
		raw, exists := p.raw[attrID]
		if !exists {
			return nil
		}
		pv = &parsedEntry{raw: raw}
		p.cache[attrID] = pv
	}
	return pv
}

func (p *ParsedExpectedValues) GetString(attrID string) (string, bool) { return getStr(p.get(attrID)) }
func (p *ParsedExpectedValues) GetStringSlice(attrID string) ([]string, bool) {
	return getStrSlice(p.get(attrID))
}
func (p *ParsedExpectedValues) GetNumber(attrID string) (float64, bool) { return getNum(p.get(attrID)) }
func (p *ParsedExpectedValues) GetNumberSlice(attrID string) ([]float64, bool) {
	return getNumSlice(p.get(attrID))
}
func (p *ParsedExpectedValues) GetBool(attrID string) (bool, bool) { return getBool(p.get(attrID)) }
func (p *ParsedExpectedValues) GetDate(attrID string) (time.Time, bool) {
	return getDate(p.get(attrID))
}
func (p *ParsedExpectedValues) GetDateBounds(attrID string) ([2]time.Time, bool) {
	return getDateBounds(p.get(attrID))
}

// ---------------------------------------------------------------------------
// Public evaluation entry points
// ---------------------------------------------------------------------------

// EvaluateRuleScore resolves the effective score for a DecisionRule by evaluating
// its RuleConditions against each Rule variation's expected attribute values
// (sourced from RuleAttributes).
//
// userAttrs carries live user attribute values (attributeID → compact JSON value).
// When non-nil, leaf conditions compare against these values.
// When nil, conditions with user-dependent attributes are treated as non-match
// (the caller is expected to defer real evaluation to delivery time).
//
// Algorithm:
//  1. No conditions → return rule.Score unchanged.
//  2. Evaluate each Rule variation in OrderNo order. Build an expected-value map
//     from the variation's RuleAttributes (attributeID → value). Return the Score
//     of the first variation whose conditions all pass.
//  3. No variation matched → return rule.Score.
func EvaluateRuleScore(rule entity.DecisionRule, userAttrs map[string]json.RawMessage) (*string, float64, error) {
	if len(rule.RuleConditions) == 0 {
		return nil, rule.Score, nil
	}

	parsed := NewParsedUserAttrs(userAttrs)

	for _, v := range sortedVariations(rule.Rules) {
		rawExpected := make(map[string]json.RawMessage, len(v.RuleAttributes))
		for _, ra := range v.RuleAttributes {
			rawExpected[ra.AttributeID.String()] = json.RawMessage(ra.Value)
		}

		pass, err := evaluateConditionGroup(rule.RuleConditions, NewParsedExpectedValues(rawExpected), parsed)
		if err != nil {
			continue
		}
		if pass {
			return &v.VariationName, float64(v.Score), nil
		}
	}

	return nil, rule.Score, nil
}

// EvaluateLogicConditions evaluates a slice of LogicCondition (from PlacementLogicEntry)
// against live user attribute values supplied in userAttrs (attr UUID → compact JSON value).
//
// It converts the LogicCondition slice into entity.RuleCondition stubs (including an
// Attribute stub that carries the DataType) and delegates to the unified
// evaluateConditionGroup chain. The "actual" value for each leaf is read from userAttrs;
// the "expected" value is taken from LogicCondition.ExpectedValue.
//
// Returns false (not an error) when a required attribute is absent from userAttrs.
func EvaluateLogicConditions(conditions []dto.LogicCondition, userAttrs map[string]json.RawMessage) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	rcs := make([]entity.RuleCondition, 0, len(conditions))
	for _, lc := range conditions {
		rcs = append(rcs, logicConditionToRuleCondition(lc))
	}

	// Only stamp entries where ExpectedValue is a non-nil, non-JSON-null value.
	// Conditions without a stamped expected value (e.g. parent/grouping nodes or
	// missing rule_attribute rows) must be skipped so that the Has guard in
	// evaluateSingleCondition returns non-match rather than comparing against a
	// zero-value produced by unmarshaling JSON null.
	rawExpected := make(map[string]json.RawMessage, len(conditions))
	for _, lc := range conditions {
		if len(lc.ExpectedValue) > 0 && string(lc.ExpectedValue) != "null" {
			rawExpected[lc.AttributeID] = lc.ExpectedValue
		}
	}

	return evaluateConditionGroup(rcs, NewParsedExpectedValues(rawExpected), NewParsedUserAttrs(userAttrs))
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

const maxConditionDepth = 3

func evaluateConditionGroup(conditions []entity.RuleCondition, expectedVals *ParsedExpectedValues, parsed *ParsedUserAttrs) (bool, error) {
	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
	}
	for k := range byParent {
		sort.Slice(byParent[k], func(i, j int) bool {
			return byParent[k][i].Sequence < byParent[k][j].Sequence
		})
	}
	roots := byParent[""]
	if len(roots) == 0 {
		return true, nil
	}
	return evalSiblings(byParent, roots, 1, expectedVals, parsed)
}

func evalSiblings(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition, depth int, expectedVals *ParsedExpectedValues, parsed *ParsedUserAttrs) (bool, error) {
	result, err := evalNode(byParent, siblings[0], depth, expectedVals, parsed)
	if err != nil {
		return false, err
	}
	for i := 1; i < len(siblings); i++ {
		c := siblings[i]
		val, err := evalNode(byParent, c, depth, expectedVals, parsed)
		if err != nil {
			return false, err
		}
		if c.ConnectorOperator == enums.ConnectorOperatorOR {
			result = result || val
		} else {
			result = result && val
		}
	}
	return result, nil
}

func evalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition, depth int, expectedVals *ParsedExpectedValues, parsed *ParsedUserAttrs) (bool, error) {
	if depth < maxConditionDepth {
		if children := byParent[c.ID.String()]; len(children) > 0 {
			return evalSiblings(byParent, children, depth+1, expectedVals, parsed)
		}
	}
	return evaluateSingleCondition(c, expectedVals, parsed)
}

func evaluateSingleCondition(c entity.RuleCondition, expectedVals *ParsedExpectedValues, parsed *ParsedUserAttrs) (bool, error) {
	attrKey := c.AttributeID.String()

	if !expectedVals.Has(attrKey) {
		return false, nil
	}
	if parsed == nil {
		return false, nil
	}
	if _, present := parsed.Raw(attrKey); !present {
		return false, nil
	}
	if c.Attribute == nil {
		return false, fmt.Errorf("condition %s: Attribute association not preloaded (need DataType)", c.ID)
	}

	return compareValuesParsed(c.Attribute.DataType, c.LogicalOperator, parsed, attrKey, expectedVals)
}

func compareValuesParsed(
	dt enums.AttributeDataType,
	op enums.LogicalOperator,
	parsed *ParsedUserAttrs,
	attrKey string,
	expectedVals *ParsedExpectedValues,
) (bool, error) {
	switch dt {
	case enums.AttributeDataTypeText:
		actual, ok := parsed.GetString(attrKey)
		if !ok {
			return false, fmt.Errorf("parse text actual value for attr %s", attrKey)
		}
		return compareTextParsed(op, actual, attrKey, expectedVals)
	case enums.AttributeDataTypeNumber:
		actual, ok := parsed.GetNumber(attrKey)
		if !ok {
			return false, fmt.Errorf("parse number actual value for attr %s", attrKey)
		}
		return compareNumberParsed(op, actual, attrKey, expectedVals)
	case enums.AttributeDataTypeDate:
		actual, ok := parsed.GetDate(attrKey)
		if !ok {
			return false, fmt.Errorf("parse date actual value for attr %s", attrKey)
		}
		return compareDateParsed(op, actual, attrKey, expectedVals)
	case enums.AttributeDataTypeBoolean:
		actual, ok := parsed.GetBool(attrKey)
		if !ok {
			return false, fmt.Errorf("parse boolean actual value for attr %s", attrKey)
		}
		return compareBooleanParsed(op, actual, attrKey, expectedVals)
	default:
		return false, fmt.Errorf("unsupported attribute data type %q", dt)
	}
}

// ---------------------------------------------------------------------------
// Comparators
// ---------------------------------------------------------------------------

func compareTextParsed(op enums.LogicalOperator, actual, attrKey string, expectedVals *ParsedExpectedValues) (bool, error) {
	switch op {
	case enums.LogicalOperatorEQ:
		exp, ok := expectedVals.GetString(attrKey)
		if !ok {
			return false, fmt.Errorf("parse text expected value for attr %s", attrKey)
		}
		return actual == exp, nil
	case enums.LogicalOperatorNEQ:
		exp, ok := expectedVals.GetString(attrKey)
		if !ok {
			return false, fmt.Errorf("parse text expected value for attr %s", attrKey)
		}
		return actual != exp, nil
	case enums.LogicalOperatorIN:
		exps, ok := expectedVals.GetStringSlice(attrKey)
		if !ok {
			return false, fmt.Errorf("parse text IN values (want JSON string array) for attr %s", attrKey)
		}
		for _, v := range exps {
			if actual == v {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("operator %q not supported for Text attribute type", op)
	}
}

func compareNumberParsed(op enums.LogicalOperator, actual float64, attrKey string, expectedVals *ParsedExpectedValues) (bool, error) {
	switch op {
	case enums.LogicalOperatorEQ, enums.LogicalOperatorNEQ,
		enums.LogicalOperatorLT, enums.LogicalOperatorLTE,
		enums.LogicalOperatorGT, enums.LogicalOperatorGTE:
		exp, ok := expectedVals.GetNumber(attrKey)
		if !ok {
			return false, fmt.Errorf("parse number expected value for attr %s", attrKey)
		}
		switch op {
		case enums.LogicalOperatorEQ:
			return actual == exp, nil
		case enums.LogicalOperatorNEQ:
			return actual != exp, nil
		case enums.LogicalOperatorLT:
			return actual < exp, nil
		case enums.LogicalOperatorLTE:
			return actual <= exp, nil
		case enums.LogicalOperatorGT:
			return actual > exp, nil
		default: // GTE
			return actual >= exp, nil
		}
	case enums.LogicalOperatorIN:
		exps, ok := expectedVals.GetNumberSlice(attrKey)
		if !ok {
			return false, fmt.Errorf("parse number IN values (want JSON number array) for attr %s", attrKey)
		}
		for _, v := range exps {
			if actual == v {
				return true, nil
			}
		}
		return false, nil
	case enums.LogicalOperatorBETWEEN:
		bounds, ok := expectedVals.GetNumberSlice(attrKey)
		if !ok {
			return false, fmt.Errorf("parse number BETWEEN bounds (want JSON number array) for attr %s", attrKey)
		}
		if len(bounds) != 2 {
			return false, fmt.Errorf("number BETWEEN expects exactly 2 bounds [min,max], got %d for attr %s", len(bounds), attrKey)
		}
		return actual >= bounds[0] && actual <= bounds[1], nil
	default:
		return false, fmt.Errorf("operator %q not supported for Number attribute type", op)
	}
}

func compareDateParsed(op enums.LogicalOperator, actual time.Time, attrKey string, expectedVals *ParsedExpectedValues) (bool, error) {
	switch op {
	case enums.LogicalOperatorEQ, enums.LogicalOperatorNEQ,
		enums.LogicalOperatorLT, enums.LogicalOperatorLTE,
		enums.LogicalOperatorGT, enums.LogicalOperatorGTE:
		exp, ok := expectedVals.GetDate(attrKey)
		if !ok {
			return false, fmt.Errorf("parse date expected value for attr %s", attrKey)
		}
		switch op {
		case enums.LogicalOperatorEQ:
			return actual.Equal(exp), nil
		case enums.LogicalOperatorNEQ:
			return !actual.Equal(exp), nil
		case enums.LogicalOperatorLT:
			return actual.Before(exp), nil
		case enums.LogicalOperatorLTE:
			return actual.Before(exp) || actual.Equal(exp), nil
		case enums.LogicalOperatorGT:
			return actual.After(exp), nil
		default: // GTE
			return actual.After(exp) || actual.Equal(exp), nil
		}
	case enums.LogicalOperatorBETWEEN:
		bounds, ok := expectedVals.GetDateBounds(attrKey)
		if !ok {
			return false, fmt.Errorf("parse date BETWEEN bounds (want [\"from\",\"to\"]) for attr %s", attrKey)
		}
		return (actual.Equal(bounds[0]) || actual.After(bounds[0])) &&
			(actual.Equal(bounds[1]) || actual.Before(bounds[1])), nil
	default:
		return false, fmt.Errorf("operator %q not supported for Date attribute type", op)
	}
}

func compareBooleanParsed(op enums.LogicalOperator, actual bool, attrKey string, expectedVals *ParsedExpectedValues) (bool, error) {
	exp, ok := expectedVals.GetBool(attrKey)
	if !ok {
		return false, fmt.Errorf("parse boolean expected value for attr %s", attrKey)
	}
	switch op {
	case enums.LogicalOperatorEQ:
		return actual == exp, nil
	case enums.LogicalOperatorNEQ:
		return actual != exp, nil
	default:
		return false, fmt.Errorf("operator %q not supported for Boolean attribute type", op)
	}
}

// ---------------------------------------------------------------------------
// Logic condition helpers
// ---------------------------------------------------------------------------

func logicConditionToRuleCondition(lc dto.LogicCondition) entity.RuleCondition {
	id, _ := uuid.Parse(lc.ConditionID)
	rc := entity.RuleCondition{
		BaseModel:         entity.BaseModel{ID: id},
		AttributeID:       mustParseUUID(lc.AttributeID),
		Sequence:          lc.Sequence,
		LogicalOperator:   enums.LogicalOperator(lc.LogicalOperator),
		ConnectorOperator: enums.ConnectorOperator(lc.ConnectorOperator),
		Attribute:         &entity.Attribute{DataType: enums.AttributeDataType(lc.DataType)},
	}
	if lc.ParentConditionID != "" {
		pid, _ := uuid.Parse(lc.ParentConditionID)
		rc.ParentRuleConditionID = &pid
	}
	return rc
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// ---------------------------------------------------------------------------
// Sorting helpers
// ---------------------------------------------------------------------------

func sortedVariations(rules []entity.Rule) []entity.Rule {
	out := make([]entity.Rule, len(rules))
	copy(out, rules)
	sort.Slice(out, func(i, j int) bool {
		return out[i].OrderNo < out[j].OrderNo
	})
	return out
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q (expected RFC3339 or YYYY-MM-DD)", s)
}
