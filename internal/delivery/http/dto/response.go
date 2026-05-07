package dto

import (
	"encoding/json"

	"kbank-ecms/internal/domain/entity/enums"
)

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

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (r *APIResponse) ToSuccess() {
	if r.Code == "" && r.Error == "" {
		r.Code = enums.SuccessResponse.String()
	}
}

func (c APIResponse) MarshalJSON() ([]byte, error) {
	// สร้าง "Alias" เพื่อเลี่ยง Infinite Loop
	type Alias APIResponse

	if c.Code == "" {
		c.ToSuccess()
	}

	// สร้าง Anonymous Struct ที่ "ฝัง" (Embed) Alias ไว้
	// แล้วกำหนด Field ที่ต้องการ "ซ่อน" ให้เป็น json:"-"
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&c),
	})
}
