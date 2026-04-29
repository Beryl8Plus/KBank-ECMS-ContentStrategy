package entity

// CustomerProfile is the decoded view of the CLEN Customer Dynamic Query
// response used by internal callers (rule evaluation, enrichment).
// Sources is keyed by datasource (collection) name; each value is the field
// map CLEN returned for that source. Keeping the value as a generic map lets
// rules look up `customer_profile.sources.<datasource>.<field>` without
// locking the schema to a fixed CLEN field set.
type CustomerProfile struct {
	CisID   string                    `json:"cisId"`
	Sources map[string]map[string]any `json:"sources"`
}
