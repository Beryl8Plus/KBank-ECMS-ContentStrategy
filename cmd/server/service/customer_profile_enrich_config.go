package service

import "time"

// CustomerProfileEnrichConfig drives the auto-enrichment of `customer_profile`
// in user_attrs. CacheTTL is the Redis TTL used when writing the enriched
// user_attrs blob back. Datasource projections are computed per-request from
// the schema registry + rules being evaluated (see extractCLENDataSources),
// so no static "Sources" list is needed here.
type CustomerProfileEnrichConfig struct {
	CacheTTL time.Duration
}
