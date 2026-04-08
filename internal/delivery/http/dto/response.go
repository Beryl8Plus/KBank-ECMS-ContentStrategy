package dto

// Pagination holds metadata for paginated list responses.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Code       string      `json:"code"`
	Error      string      `json:"error,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}
