package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// AppConfig is the root configuration struct for all services.
// Values are loaded from a YAML file first, then overridden by environment variables.
// Priority order: ENV > YAML value > env-default tag.
type AppConfig struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Env    string        `yaml:"env"    env:"SETENV"`
	Port   string        `yaml:"port"   env:"PORT"           env-default:"8081"`
	Config ServiceConfig `yaml:"config"`
}

type ServiceConfig struct {
	Timeout   TimeoutConfig     `yaml:"timeout"`
	Postgres  PostgresConfig    `yaml:"postgres"`
	Redis     RedisConfig       `yaml:"redis"`
	Cache     CacheConfig       `yaml:"cache"`
	Swagger   SwaggerConfig     `yaml:"swagger"`
	RateLimit RateLimitConfig   `yaml:"rate_limit"`
	CLENLead  CLENLeadAPIConfig `yaml:"clen_lead"`
}

// CLENLeadAPIConfig holds connection settings for the CLEN Lead API.
// Credentials (APIKey, AppIdentifier) must always be supplied via ENV vars and never committed to YAML.
type CLENLeadAPIConfig struct {
	BaseURL       string        `yaml:"base_url"       env:"CLEN_LEAD_API_BASE_URL"`
	Path          string        `yaml:"path"           env:"CLEN_LEAD_API_PATH"`
	APIKey        string        `yaml:"api_key"        env:"CLEN_LEAD_API_KEY"`
	AppIdentifier string        `yaml:"app_identifier" env:"CLEN_LEAD_APP_ID"`
	Timeout       time.Duration `yaml:"timeout"        env:"CLEN_LEAD_API_TIMEOUT" env-default:"5s"`
	RetryCount    int           `yaml:"retry_count"    env:"CLEN_LEAD_API_RETRIES"  env-default:"2"`
	ExpireFilter  string        `yaml:"expire_filter"  env:"CLEN_LEAD_EXP_F"        env-default:"true"`
}

// PostgresConfig holds PostgreSQL connection details.
// Credentials are always supplied via ENV; YAML provides non-secret defaults.
type PostgresConfig struct {
	Host     string `yaml:"host"     env:"DB_HOST"     env-default:"localhost"`
	Port     string `yaml:"port"     env:"DB_PORT"     env-default:"5432"`
	User     string `yaml:"user"     env:"DB_USER"     env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name"  env:"DB_NAME"     env-default:"kbank_ecms"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSLMODE"  env-default:"disable"`
}

// RedisConfig holds Redis connection details.
// PrincipalID is required only when SETENV != DEVLOCAL (Azure Workload Identity).
type RedisConfig struct {
	Host        string `yaml:"host"         env:"REDIS_HOST"          env-default:"localhost"`
	Port        string `yaml:"port"         env:"REDIS_PORT"          env-default:"6379"`
	Username    string `yaml:"username"     env:"REDIS_USERNAME"` // Only needed for DEVGCP; ignored otherwise
	Password    string `yaml:"password"     env:"REDIS_PASSWORD"`
	TLS         bool   `yaml:"tls"          env:"REDIS_TLS"           env-default:"false"` // DEVGCP: set true only when Memorystore in-transit encryption is enabled
	PrincipalID string `yaml:"principal_id" env:"REDIS_PRINCIPAL_ID"`
}

type CacheConfig struct {
	TTL             time.Duration `yaml:"ttl"              env:"CMS_RUNTIME_TTL"      env-default:"15m"`
	RefreshInterval time.Duration `yaml:"refresh_interval" env:"CMS_RUNTIME_INTERVAL" env-default:"5m"`
}

type SwaggerConfig struct {
	Host string `yaml:"host" env:"SWAGGER_HOST"`
}

type TimeoutConfig struct {
	ReqCtxTimeout time.Duration `yaml:"req_ctx_timeout" env-default:"30s"`
	DBCtxTimeout  time.Duration `yaml:"db_ctx_timeout"  env-default:"10s"`
}

// RateLimitConfig holds inbound rate-limiting parameters for the HTTP server.
// These can be overridden per-environment via the YAML file; no ENV override is needed.
type RateLimitConfig struct {
	RPS   int `yaml:"requests_per_second"     env-default:"50"`
	Burst int `yaml:"burst"                   env-default:"100"`
	MCR   int `yaml:"max_concurrent_requests" env-default:"10"`
}

// JWTConfig holds JWT signing configuration.
// Secret must be supplied via ENV; never commit to YAML.
type JWTConfig struct {
	Secret string        `yaml:"secret"   env:"JWT_SECRET_KEY"      env-default:"change-me-in-production"`
	Expiry time.Duration `yaml:"expiry"   env:"JWT_TOKEN_DURATION"  env-default:"24h"`
	Issuer string        `yaml:"issuer"   env:"JWT_ISSUER"          env-default:"kbank-ecms"`
}

// LoadConfig reads the YAML file at path and applies ENV overrides automatically.
// Pass the service-specific config path, e.g. "configs/delivery.yaml".
func LoadConfig(path string) (AppConfig, error) {
	var cfg AppConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("loading config from %q: %w", path, err)
	}
	return cfg, nil
}
