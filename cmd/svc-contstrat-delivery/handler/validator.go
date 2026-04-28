package handler

import (
	"fmt"
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

	// customer_id_format is a field-level custom tag that applies conditional
	// format rules based on the sibling CustomerIDType field:
	//   - CIS_ID / IP_ID        → customerId must be exactly 10 decimal digits.
	//   - KPLUS_MOBILE_NUMBER / LINE_UUID → any non-empty value is accepted
	//     (the `required` binding tag already enforces non-empty).
	//
	// Usage in a struct:
	//   CustomerID string `form:"customerId" binding:"required,customer_id_format"`
	if err := v.RegisterValidation("customer_id_format", validateCustomerIDFormat); err != nil {
		panic("validator: failed to register customer_id_format: " + err.Error())
	}
}

// validateCustomerIDFormat is the field-level validator for the
// "customer_id_format" binding tag.
//
// It inspects the sibling CustomerIDType field on the parent struct to decide
// which format rule to apply:
//   - Empty string always passes — the `required` tag handles that separately.
//   - CIS_ID / IP_ID           → value must match [0-9]{10}.
//   - KPLUS_MOBILE_NUMBER / LINE_UUID → any non-empty value is accepted.
func validateCustomerIDFormat(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true // defer to `required`
	}

	// Read the sibling CustomerIDType from the parent struct.
	parent := fl.Parent()
	idTypeField := parent.FieldByName("CustomerIDType")
	if !idTypeField.IsValid() {
		// Struct does not carry CustomerIDType; skip format enforcement.
		return true
	}

	switch dto.CustomerIdType(idTypeField.String()) {
	case dto.CustomerIdTypeCISID, dto.CustomerIdTypeIPID:
		return tenDigitNumeric.MatchString(val)
	default:
		// KPLUS_MOBILE_NUMBER, LINE_UUID — any non-empty value is valid.
		return true
	}
}

func MessageTranslator(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "oneof":
		return fmt.Sprintf("This field must be one of: %s", fe.Param())
	case "required_if":
		return "This field is required when the specified condition is met"
	case "min":
		return fmt.Sprintf("At least %s item,length(s) are required", fe.Param())
	case "max":
		return fmt.Sprintf("At most %s item,length(s) are allowed", fe.Param())
	case "numeric":
		return "This field must be a numeric string"
	case "len":
		return fmt.Sprintf("This field must be exactly %s characters long", fe.Param())
	case "customer_id_format":
		return "Must be a 10-digit numeric string"
	}
	return fe.Error() // default error
}
