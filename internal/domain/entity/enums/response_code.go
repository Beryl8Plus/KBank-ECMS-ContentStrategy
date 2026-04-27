package enums

type ResponseCode string

const (
	SuccessResponse        ResponseCode = "SUCCESS"
	ErrorCodeBadRequest    ResponseCode = "ERROR_BAD_REQUEST"
	ErrorCodeInternalError ResponseCode = "ERROR_INTERNAL_ERROR"
	// Add more error codes as needed
	// ErrorCodeNotFound     ResponseCode = "ERROR_NOT_FOUND"
)

func (e ResponseCode) String() string {
	return string(e)
}
