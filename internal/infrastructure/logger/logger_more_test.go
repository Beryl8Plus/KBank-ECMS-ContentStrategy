package logger_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

func TestLReqResClient_Response(t *testing.T) {
	rlog := entity.RequestLog{
		Service:  "svc",
		Status:   "200",
		Duration: "5ms",
	}
	out := captureStdout(func() { logger.LReqResClient(context.Background(), rlog, true) })
	if !strings.Contains(out, "RESPONSE") || !strings.Contains(out, "5ms") {
		t.Errorf("response log missing fields: %q", out)
	}
}

func TestLReqResApp_BothBranches(t *testing.T) {
	rlog := entity.ResponseLog{
		From:          "delivery",
		To:            "redis",
		CorrelationID: "cid",
		Body:          "GET key",
		Key:           "k1",
		Status:        "OK",
		Duration:      "1ms",
	}
	reqOut := captureStdout(func() { logger.LReqResApp(context.Background(), rlog, false) })
	if !strings.Contains(reqOut, "APP-REQUEST") || !strings.Contains(reqOut, "redis") {
		t.Errorf("APP-REQUEST log missing fields: %q", reqOut)
	}
	respOut := captureStdout(func() { logger.LReqResApp(context.Background(), rlog, true) })
	if !strings.Contains(respOut, "APP-RESPONSE") || !strings.Contains(respOut, "OK") {
		t.Errorf("APP-RESPONSE log missing fields: %q", respOut)
	}
}

func TestLResponse(t *testing.T) {
	out := captureStdout(func() {
		logger.LResponse(context.Background(), entity.ResponseLog{From: "a", To: "b", Status: "OK"})
	})
	if !strings.Contains(out, "APP-RESPONSE") {
		t.Errorf("LResponse should produce APP-RESPONSE log, got %q", out)
	}
}

func TestLSystem_AllBranches(t *testing.T) {
	out := captureStdout(func() {
		logger.LSystem(context.Background(), entity.SystemLog{
			Level:       "INFO",
			Service:     "svc",
			Message:     "hello",
			Action:      "start",
			EnvDetail:   "DEVLOCAL",
			SystemEvent: "BOOT",
		})
	})
	if !strings.Contains(out, "hello") || !strings.Contains(out, "DEVLOCAL") || !strings.Contains(out, "BOOT") {
		t.Errorf("LSystem missing fields: %q", out)
	}
}

func TestLSystem_ActionWithoutMessage(t *testing.T) {
	out := captureStdout(func() {
		logger.LSystem(context.Background(), entity.SystemLog{
			Level:   "INFO",
			Service: "svc",
			Action:  "alone",
		})
	})
	if !strings.Contains(out, "alone") {
		t.Errorf("LSystem with only Action missing it: %q", out)
	}
}

func TestLSystem_BareLine(t *testing.T) {
	out := captureStdout(func() {
		logger.LSystem(context.Background(), entity.SystemLog{Level: "INFO", Service: "svc"})
	})
	if !strings.Contains(out, "svc") {
		t.Errorf("LSystem bare line missing svc: %q", out)
	}
}

func TestLStartup_AllBranches(t *testing.T) {
	for _, c := range []struct {
		name string
		log  entity.StartupLog
		want string
	}{
		{"message", entity.StartupLog{Level: "INFO", Service: "boot", Message: "msg"}, "msg"},
		{"detail", entity.StartupLog{Level: "INFO", Service: "boot", Detail: "det"}, "det"},
		{"config", entity.StartupLog{Level: "INFO", Service: "boot", Config: "cfg"}, "Config snippet: cfg"},
		{"env", entity.StartupLog{Level: "INFO", Service: "boot", EnvDetail: "DEV"}, "Current ENV: DEV"},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			out := captureStderr(func() { logger.LStartup(context.Background(), c.log) })
			if !strings.Contains(out, c.want) {
				t.Errorf("LStartup %s: missing %q in %q", c.name, c.want, out)
			}
		})
	}
}

func TestPopulateErrorLog_FillsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/oops?x=1", nil)
	c.Request.Header.Set("X-Workflow-ID", "wf-1")
	c.Request.Header.Set("requestID", "corr-1")

	err := errors.New("boom")
	got := logger.PopulateErrorLog(c, nil, "svc", "{body}", err, map[string]string{
		"error_code":  "E001",
		"customer_id": "cust-1",
		"employee_id": "emp-1",
		"env":         "dev",
	})
	if got.Service != "svc" || got.ErrorCode != "E001" || got.Message != "boom" ||
		got.WorkflowID != "wf-1" || got.CustomerID != "cust-1" || got.EmployeeID != "emp-1" ||
		got.Env != "dev" || got.RequestURI != "/oops?x=1" || got.CorrelationID != "corr-1" ||
		got.LogID == "" || got.StackTrace == "" {
		t.Errorf("unexpected ErrorLog: %+v", got)
	}
}

func TestPopulateErrorLog_PreservesExistingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/x", nil)

	in := &entity.ErrorLog{
		Service: "preset", ErrorCode: "preE", Message: "preMsg",
		WorkflowID: "preW", CustomerID: "preC", EmployeeID: "preE2", Env: "preEnv",
		Request: "preReq", ClientIP: "1.1.1.1", CorrelationID: "preCid", LogID: "preLog",
	}
	got := logger.PopulateErrorLog(c, in, "svc", "body", fmt.Errorf("err"), map[string]string{})
	if got.Service != "preset" || got.ErrorCode != "preE" || got.Message != "preMsg" ||
		got.WorkflowID != "preW" || got.CustomerID != "preC" || got.EmployeeID != "preE2" ||
		got.Env != "preEnv" || got.Request != "preReq" || got.ClientIP != "1.1.1.1" ||
		got.CorrelationID != "preCid" || got.LogID != "preLog" {
		t.Errorf("ErrorLog should preserve preset fields: %+v", got)
	}
}

func TestPopulateErrorLog_NilContext(t *testing.T) {
	got := logger.PopulateErrorLog(nil, nil, "svc", "", nil, map[string]string{
		"workflow_id":    "wf",
		"correlation_id": "cor",
	})
	if got.Service != "svc" || got.WorkflowID != "wf" || got.CorrelationID != "cor" || got.LogID == "" {
		t.Errorf("nil ctx fallback failed: %+v", got)
	}
}
