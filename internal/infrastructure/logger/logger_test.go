package logger_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	f()
	w.Close()
	os.Stdout = orig
	return <-outC
}

func captureStderr(f func()) string {
	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	f()
	w.Close()
	os.Stderr = orig
	return <-outC
}

func TestLRequestWritesExpectedFields(t *testing.T) {
	rlog := entity.RequestLog{
		Level:           "INFO",
		Service:         "my-service",
		Method:          "GET",
		URL:             "/test",
		ClientIP:        "127.0.0.1",
		Message:         "hello",
		Headers:         "h:1",
		Status:          "200",
		Duration:        "10ms",
		RequestPayload:  "req",
		ResponsePayload: "resp",
	}

	out := captureStdout(func() { logger.LRequest(context.Background(), rlog) })

	if !strings.Contains(out, "my-service") || !strings.Contains(out, "GET") || !strings.Contains(out, "/test") || !strings.Contains(out, "hello") {
		t.Fatalf("unexpected stdout: %q", out)
	}
}

func TestLErrorWritesToStderr(t *testing.T) {
	elog := entity.ErrorLog{
		Service:   "err-svc",
		ErrorCode: "E001",
		Message:   "something failed",
	}

	out := captureStderr(func() { logger.LError(context.Background(), elog) })

	if !strings.Contains(out, "E001") || !strings.Contains(out, "something failed") || !strings.Contains(out, "err-svc") {
		t.Fatalf("unexpected stderr: %q", out)
	}
}
