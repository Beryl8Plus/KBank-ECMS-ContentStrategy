package model

type ErrorLog struct {
	Service       string
	ErrorCode     string
	Message       string
	WorkflowID    string
	CustomerID    string
	EmployeeID    string
	Env           string
	RequestURI    string
	Request       string
	ClientIP      string
	CorrelationID string
	StackTrace    string
	LogID         string
}

type RequestLog struct {
	Service         string
	Level           string // REQUEST / FORWARD / RESPONSE
	Method          string
	URL             string
	ClientIP        string
	CorrelationID   string
	Headers         string
	Status          string
	Message         string
	Duration        string
	RequestPayload  string
	ResponsePayload string
	Context         string
	StackTrace      string
	LogID           string
}

type SystemLog struct {
	Service       string
	Level         string // INFO / DEBUG / WARN
	Message       string
	CorrelationID string
	Action        string
	EnvDetail     string
	SystemEvent   string
}

type StartupLog struct {
	Service   string
	Level     string // ERROR / DEBUG
	Message   string
	Detail    string
	Config    string
	EnvDetail string
}

type ResponseLog struct {
	Service       string
	From          string
	To            string
	Status        string
	CorrelationID string
	Duration      string
	Body          string
	Key           string
}
