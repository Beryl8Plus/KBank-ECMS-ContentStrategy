package pubsub

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stubRedis records Publish calls and returns a configured error.
type stubRedis struct {
	publishCalls int
	lastChannel  string
	lastPayload  string
	publishErr   error
}

func (s *stubRedis) Publish(_ context.Context, channel, payload string) error {
	s.publishCalls++
	s.lastChannel = channel
	s.lastPayload = payload
	return s.publishErr
}

// satisfy the rest of the RedisCacheRepository interface — none of these
// are called by Publisher.PingSync.
func (s *stubRedis) Get(context.Context, string) (string, error)            { return "", nil }
func (s *stubRedis) Set(context.Context, string, string, time.Duration) error { return nil }
func (s *stubRedis) HGet(context.Context, string, string) (string, error)   { return "", nil }
func (s *stubRedis) HSet(context.Context, string, string, string) error     { return nil }
func (s *stubRedis) FlushDB(context.Context) error                          { return nil }
func (s *stubRedis) GetSet(_ context.Context, _ string, _ time.Duration, _ func(context.Context) (string, error)) (string, error) {
	return "", nil
}
func (s *stubRedis) Delete(context.Context, string) error { return nil }
func (s *stubRedis) Subscribe(context.Context, string) (<-chan string, error) {
	return make(chan string), nil
}

func TestPingSync_NilReceiverNoOp(t *testing.T) {
	t.Parallel()
	var p *Publisher
	if err := p.PingSync(context.Background(), "hero", "rule-1", "v1"); err != nil {
		t.Errorf("nil receiver should no-op, got %v", err)
	}
}

func TestPingSync_NilRedisNoOp(t *testing.T) {
	t.Parallel()
	p := NewPublisher(nil)
	if err := p.PingSync(context.Background(), "hero", "rule-1", "v1"); err != nil {
		t.Errorf("nil redis should no-op, got %v", err)
	}
}

func TestPingSync_HappyPath(t *testing.T) {
	t.Parallel()
	r := &stubRedis{}
	p := NewPublisher(r)

	err := p.PingSync(context.Background(), "hero", "rule-1", "v1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.publishCalls != 1 {
		t.Errorf("Publish should be called once, got %d", r.publishCalls)
	}
	if r.lastChannel != ChannelCMSSyncPing {
		t.Errorf("channel = %q, want %q", r.lastChannel, ChannelCMSSyncPing)
	}
	// payload is JSON-marshalled SyncPingMessage — quick spot check
	for _, want := range []string{`"placement_name":"hero"`, `"version_hash":"v1"`, `"decision_rule_id":"rule-1"`} {
		if !contains(r.lastPayload, want) {
			t.Errorf("payload missing %s — got %s", want, r.lastPayload)
		}
	}
}

func TestPingSync_OmitsEmptyDecisionRuleID(t *testing.T) {
	t.Parallel()
	r := &stubRedis{}
	_ = NewPublisher(r).PingSync(context.Background(), "hero", "", "v1")
	// JSON tag uses omitempty
	if contains(r.lastPayload, "decision_rule_id") {
		t.Errorf("empty decision_rule_id should be omitted, got %s", r.lastPayload)
	}
}

func TestPingSync_RedisErrorReturned(t *testing.T) {
	t.Parallel()
	r := &stubRedis{publishErr: errors.New("redis down")}
	p := NewPublisher(r)
	err := p.PingSync(context.Background(), "hero", "", "")
	if err == nil {
		t.Fatal("expected error when redis fails")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
