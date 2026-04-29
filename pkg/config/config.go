package config

import (
	"time"
)

type AppConfig struct {
	AppName   string          `json:"appName"`
	Endpoint  string          `yaml:"endpoint"`
	Timeout   TimeoutConfig   `json:"timeout"`
	RateLimit RateLimitConfig `json:"rateLimit"`
	// Add other configuration fields as needed (e.g., CORS settings, logging options, etc.)
}

// InboundConfig represents the YAML structure for inbound service configuration.
type InboundConfig struct {
	Server []Server `yaml:"server"`
}

// Server represents a single server entry in the inbound configuration.
type Server struct {
	Name      string          `yaml:"name"`
	Endpoint  string          `yaml:"endpoint"`
	RateLimit RateLimitConfig `yaml:"ratelimit"`
}

// RateLimitConfig holds rate-limiting configuration for a server.
type RateLimitConfig struct {
	RPS   int `yaml:"requests_per_second"`
	Burst int `yaml:"burst"`
	MCR   int `yaml:"max_concurrent_requests"`
}

// RedisConfig holds connection details for a Redis instance.
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

// PostgresConfig holds connection details for a PostgreSQL instance.
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type TimeoutConfig struct {
	ReqCtxTimeout time.Duration `default:"30s" json:"reqCtxTimeout"`
	DBCtxTimeout  time.Duration `default:"10s" json:"dbCtxTimeout"`
}

func NewAppConfig() AppConfig {
	rateLimitCfg := RateLimitConfig{RPS: 50, Burst: 100, MCR: 10}
	if cfgRateLimit, err := LoadNewServiceRateLimit("./configs/newservice_inbound_config.yaml"); err == nil {
		rateLimitCfg = cfgRateLimit
	}

	timeoutCfg := TimeoutConfig{ReqCtxTimeout: 30 * time.Second, DBCtxTimeout: 10 * time.Second}

	// In a real application, you would load these from environment variables, config files, or a config service.
	// For this example, we'll hardcode some values.
	return AppConfig{
		AppName: "cms-delivery",
		Timeout: TimeoutConfig{
			ReqCtxTimeout: timeoutCfg.ReqCtxTimeout,
			DBCtxTimeout:  timeoutCfg.DBCtxTimeout,
		},
		RateLimit: RateLimitConfig{
			RPS:   rateLimitCfg.RPS,   // Allow 50 requests per second
			Burst: rateLimitCfg.Burst, // Allow bursts of up to 100 requests
			MCR:   rateLimitCfg.MCR,   // Allow up to 10 concurrent requests
		},
	}
}
