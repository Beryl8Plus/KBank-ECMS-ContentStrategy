package util

import (
	"fmt"
	"kbank-ecms/internal/model"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	mu sync.Mutex
)

func init() {
	// Logging is now directed to stdout/stderr
}

func formatTime() string {
	return time.Now().Format("2006-01-02 15:04:05|")
}

func generateLogID() string {
	return uuid.New().String()
}

func LError(e model.ErrorLog) {
	mu.Lock()
	defer mu.Unlock()

	id := e.LogID
	if id == "" {
		id = generateLogID()
	}

	logLine := fmt.Sprintf("%slog_id: %s|ERROR|service_module: %s|client_ip: %s|correlation_id: %s|url: %s|request_payload: %s|error_code: %s|message: %s|stack_trace: %s\n",
		formatTime(),
		id,
		e.Service,
		e.ClientIP,
		e.CorrelationID,
		e.RequestURI,
		e.Request,
		e.ErrorCode,
		e.Message,
		e.StackTrace,
	)

	fmt.Fprint(os.Stderr, logLine)
}

func LReqResClient(r model.RequestLog, isResponse bool) {
	mu.Lock()
	defer mu.Unlock()

	id := r.LogID
	if id == "" {
		id = generateLogID()
	}

	if !isResponse {
		logLine := fmt.Sprintf("%slog_id:%s|REQUEST|service_module:%s|method:%s|url:%s|client_ip:%s|correlation_id:%s|headers:%s|request_payload:%s|context:%s\n", formatTime(), id, r.Service, r.Method, r.URL, r.ClientIP, r.CorrelationID, r.Headers, r.RequestPayload, r.Context)
		fmt.Fprint(os.Stdout, logLine)
		return
	}

	// response
	logLine := fmt.Sprintf("%slog_id:%s|RESPONSE|service_module:%s|correlation_id:%s|status_code:%s|duration:%s|response_payload:%s|context:%s\n", formatTime(), id, r.Service, r.CorrelationID, r.Status, r.Duration, r.ResponsePayload, r.Context)
	fmt.Fprint(os.Stdout, logLine)
}

// LRequest is a convenience wrapper to log incoming client requests.
func LRequest(r model.RequestLog) {
	LReqResClient(r, false)
}

func LReqResApp(r model.ResponseLog, isResponse bool) {
	mu.Lock()
	defer mu.Unlock()

	if !isResponse {
		logLine := fmt.Sprintf("%sAPP-REQUEST|%s|%s → %s Body: %s|Key: %s\n", formatTime(), r.CorrelationID, r.From, r.To, r.Body, r.Key)
		fmt.Print(logLine)
		return
	}

	logLine := fmt.Sprintf("%sAPP-RESPONSE|%s|%s → %s Status: %s|Duration: %s|Body: %s|Key: %s\n", formatTime(), r.CorrelationID, r.From, r.To, r.Status, r.Duration, r.Body, r.Key)
	fmt.Print(logLine)
}

// LResponse is a compatibility wrapper used across the codebase to log
// application-level request/response events (e.g., Redis/AEM interactions).
func LResponse(r model.ResponseLog) {
	// default to response-style log
	LReqResApp(r, true)
}

func LSystem(s model.SystemLog) {
	mu.Lock()
	defer mu.Unlock()

	base := fmt.Sprintf("%s%s|%s|%s|", formatTime(), s.Level, s.CorrelationID, s.Service)
	var logLine string

	if s.Message != "" {
		base += " " + s.Message
	}

	var details []string
	if s.Action != "" {
		if s.Message == "" {
			base += " " + s.Action
		} else {
			details = append(details, s.Action)
		}
	}

	if s.EnvDetail != "" {
		details = append(details, "|ENV: "+s.EnvDetail)
	}

	if s.SystemEvent != "" {
		details = append(details, "|"+s.SystemEvent)
	}

	if len(details) > 0 {
		logLine = fmt.Sprintf("%s %s\n", base, strings.Join(details, " | "))
	} else {
		logLine = base + "\n"
	}

	fmt.Print(logLine)
}

func LStartup(s model.StartupLog) {
	mu.Lock()
	defer mu.Unlock()

	base := fmt.Sprintf("%s%s|%s|", formatTime(), s.Level, s.Service)
	var content string

	if s.Message != "" {
		content = s.Message
	} else if s.Detail != "" {
		content = s.Detail
	} else if s.Config != "" {
		content = "Config snippet: " + s.Config
	} else if s.EnvDetail != "" {
		content = "Current ENV: " + s.EnvDetail
	}

	logLine := fmt.Sprintf("%s %s\n", base, content)

	fmt.Fprint(os.Stderr, logLine)
}

// PopulateErrorLog fills missing fields in an existing ErrorLog using request/context info.
// It preserves fields already set by the caller (e.g., ErrorCode, Message) and only fills
// defaults where values are empty.
func PopulateErrorLog(c *gin.Context, e *model.ErrorLog, svc, reqBody string, err error, extra map[string]string) model.ErrorLog {
	if e == nil {
		var tmp model.ErrorLog
		e = &tmp
	}

	if e.Service == "" {
		e.Service = svc
	}

	if e.RequestURI == "" && c != nil {
		e.RequestURI = c.Request.RequestURI
	}

	if e.ErrorCode == "" {
		if v, ok := extra["error_code"]; ok {
			e.ErrorCode = v
		}
	}

	if e.Message == "" && err != nil {
		e.Message = err.Error()
	}

	if e.WorkflowID == "" {
		if c != nil {
			e.WorkflowID = c.GetHeader("X-Workflow-ID")
		}
		if e.WorkflowID == "" {
			if v, ok := extra["workflow_id"]; ok {
				e.WorkflowID = v
			}
		}
	}

	if e.CustomerID == "" {
		if v, ok := extra["customer_id"]; ok {
			e.CustomerID = v
		}
	}

	if e.EmployeeID == "" {
		if v, ok := extra["employee_id"]; ok {
			e.EmployeeID = v
		}
	}

	if e.Env == "" {
		if v, ok := extra["env"]; ok {
			e.Env = v
		}
	}

	if e.Request == "" {
		e.Request = reqBody
	}

	if e.ClientIP == "" && c != nil {
		e.ClientIP = c.ClientIP()
	}

	if e.CorrelationID == "" {
		if c != nil {
			e.CorrelationID = c.GetHeader("requestID")
		}
		if e.CorrelationID == "" {
			if v, ok := extra["correlation_id"]; ok {
				e.CorrelationID = v
			} else {
				e.CorrelationID = uuid.New().String()
			}
		}
	}

	// Always attach latest stacktrace for the error occurrence
	if err != nil {
		e.StackTrace = fmt.Sprintf("%v", err)
	}

	if e.LogID == "" {
		e.LogID = uuid.New().String()
	}

	return *e
}
