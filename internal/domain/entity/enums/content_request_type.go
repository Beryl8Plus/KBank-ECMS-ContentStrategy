package enums

// RequestType is the enumeration of valid content request types.
type RequestType string

const (
	RequestTypePersonalizedContent RequestType = "personalizedContent"
	RequestTypeStaticContent       RequestType = "staticContent"
	RequestTypeArticleCategory     RequestType = "articleCategory"
)

// IsValid reports whether rt is a recognised RequestType value.
func (rt RequestType) IsValid() bool {
	switch rt {
	case RequestTypePersonalizedContent, RequestTypeStaticContent, RequestTypeArticleCategory:
		return true
	}
	return false
}
