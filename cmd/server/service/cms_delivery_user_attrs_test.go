package service

// All previously-here lead-enrichment tests covered behavior of
// enrichLeadOffering, which was removed alongside the per-datasource
// cache + UUID-key transform refactor. resolveUserAttrs no longer touches
// lead_offering at all; SALES_TARGET lead enrichment is exercised by
// TestFetchLeadsForRequest_* in cms_delivery_lead_test.go.
