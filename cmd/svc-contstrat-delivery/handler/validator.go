package handler

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"kbank-ecms/internal/delivery/http/dto"
)

// tenDigitNumeric matches exactly 10 ASCII decimal digits.
var tenDigitNumeric = regexp.MustCompile(`^[0-9]{10}$`)

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	// Report validation errors using the form tag name (e.g. "customerId") instead
	// of the Go struct field name (e.g. "CustomerID"), so error messages are
	// consistent with the query-param names clients send.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})

	v.RegisterStructValidation(validateContentRequest, dto.ContentRequestQueryParams{})
}

// validateContentRequest enforces cross-field rules for ContentRequestQueryParams.
// Field-level binding tags handle required/oneof; this function adds format rules:
//   - CIS_ID / IP_ID: customerId must be exactly 10 decimal digits.
//   - KPLUS_MOBILE_NUMBER / LINE_UUID: any non-empty value is accepted (required is
//     already enforced by the binding tag).
func validateContentRequest(sl validator.StructLevel) {
	req := sl.Current().Interface().(dto.ContentRequestQueryParams)

	switch req.CustomerIDType {
	case dto.CustomerIdTypeCISID, dto.CustomerIdTypeIPID:
		if req.CustomerID != "" && !tenDigitNumeric.MatchString(req.CustomerID) {
			sl.ReportError(req.CustomerID, "customerId", "CustomerID", "customer_id_format", "")
		}
	}
}
