package dto

// CustomerIdType identifies the scheme used for customerId.
type CustomerIdType string

const (
	CustomerIdTypeCISID CustomerIdType = "CIS_ID"
)

// FlushRequest is the request body for POST /flush.
type FlushRequest struct {
	Placements []string `json:"placements"`
	IsEvaluate bool     `json:"isEvaluate"` // Optional flag to trigger cache re-population after flush
}

type FlushResponse struct {
	Message string `json:"message"`
}
