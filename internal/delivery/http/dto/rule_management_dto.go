package dto

// RuleManagementRequest is the request body for the Rule Management API.
type RuleManagementRequest struct {
	// Extend with fields as the feature is implemented.
}

// RuleManagementResponse is the response body for the Rule Management API.
type RuleManagementResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
