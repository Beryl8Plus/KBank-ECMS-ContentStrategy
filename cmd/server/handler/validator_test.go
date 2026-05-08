package handler

import (
	"testing"

	"github.com/go-playground/validator/v10"

	"kbank-ecms/internal/delivery/http/dto"
)

func TestValidateCustomerIDFormat_ViaStructValidation(t *testing.T) {
	v := validator.New()
	if err := v.RegisterValidation("customer_id_format", validateCustomerIDFormat); err != nil {
		t.Fatalf("register: %v", err)
	}

	type req struct {
		CustomerID     string             `validate:"customer_id_format"`
		CustomerIDType dto.CustomerIdType `validate:"-"`
	}

	cases := []struct {
		name    string
		val     req
		wantErr bool
	}{
		{"empty defers to required", req{CustomerID: "", CustomerIDType: dto.CustomerIdTypeCISID}, false},
		{"CIS_ID 10 digits", req{CustomerID: "0000000001", CustomerIDType: dto.CustomerIdTypeCISID}, false},
		{"CIS_ID 9 digits", req{CustomerID: "000000001", CustomerIDType: dto.CustomerIdTypeCISID}, true},
		{"CIS_ID alpha", req{CustomerID: "abcdefghij", CustomerIDType: dto.CustomerIdTypeCISID}, true},
		{"IP_ID 10 digits", req{CustomerID: "1234567890", CustomerIDType: dto.CustomerIdTypeIPID}, false},
		{"KPLUS any non-empty ok", req{CustomerID: "08123", CustomerIDType: dto.CustomerIdTypeKPlusMobileNumber}, false},
		{"LINE any non-empty ok", req{CustomerID: "uuid-xxx", CustomerIDType: dto.CustomerIdTypeLineUUID}, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := v.Struct(c.val)
			if (err != nil) != c.wantErr {
				t.Errorf("err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}

// TestMessageTranslator_AllTags exercises every branch of the translator by
// constructing structs that fail real validator tags; we then unwrap each
// validator.ValidationErrors entry and feed it through MessageTranslator.
func TestMessageTranslator_AllTags(t *testing.T) {
	v := validator.New()
	if err := v.RegisterValidation("customer_id_format", validateCustomerIDFormat); err != nil {
		t.Fatalf("register: %v", err)
	}

	cases := []struct {
		name string
		make func() error
		want string
	}{
		{
			"required",
			func() error {
				type s struct {
					Field string `validate:"required"`
				}
				return v.Struct(s{})
			},
			"This field is required",
		},
		{
			"oneof",
			func() error {
				type s struct {
					Field string `validate:"oneof=a b c"`
				}
				return v.Struct(s{Field: "z"})
			},
			"This field must be one of: a b c",
		},
		{
			"min",
			func() error {
				type s struct {
					Field string `validate:"min=3"`
				}
				return v.Struct(s{Field: "ab"})
			},
			"At least 3 item,length(s) are required",
		},
		{
			"max",
			func() error {
				type s struct {
					Field string `validate:"max=2"`
				}
				return v.Struct(s{Field: "abcd"})
			},
			"At most 2 item,length(s) are allowed",
		},
		{
			"numeric",
			func() error {
				type s struct {
					Field string `validate:"numeric"`
				}
				return v.Struct(s{Field: "abc"})
			},
			"This field must be a numeric string",
		},
		{
			"len",
			func() error {
				type s struct {
					Field string `validate:"len=4"`
				}
				return v.Struct(s{Field: "ab"})
			},
			"This field must be exactly 4 characters long",
		},
		{
			"customer_id_format",
			func() error {
				type s struct {
					CustomerID     string             `validate:"customer_id_format"`
					CustomerIDType dto.CustomerIdType `validate:"-"`
				}
				return v.Struct(s{CustomerID: "abcd", CustomerIDType: dto.CustomerIdTypeCISID})
			},
			"Must be a 10-digit numeric string",
		},
		{
			"required_if",
			func() error {
				type s struct {
					Type   string `validate:"-"`
					Detail string `validate:"required_if=Type X"`
				}
				return v.Struct(s{Type: "X"})
			},
			"This field is required when the specified condition is met",
		},
		{
			"unknown-tag fallthrough",
			func() error {
				type s struct {
					Field string `validate:"email"`
				}
				return v.Struct(s{Field: "not-an-email"})
			},
			"", // we'll just check that translator returns non-empty (default fe.Error())
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := c.make()
			if err == nil {
				t.Fatal("expected validation error")
			}
			ve, ok := err.(validator.ValidationErrors)
			if !ok || len(ve) == 0 {
				t.Fatalf("expected ValidationErrors, got %T", err)
			}
			got := MessageTranslator(ve[0])
			if c.want == "" {
				if got == "" {
					t.Error("translator should return non-empty for unknown tag")
				}
				return
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
