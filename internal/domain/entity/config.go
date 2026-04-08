package entity

// InboundConfig represents the YAML structure for inbound service configuration.
type InboundConfig struct {
	Server []Server `yaml:"server"`
}

// Server represents a single server entry in the inbound configuration.
type Server struct {
	Name      string    `yaml:"name"`
	Endpoint  string    `yaml:"endpoint"`
	RateLimit RateLimit `yaml:"ratelimit"`
}

// RateLimit holds rate-limiting configuration for a server.
type RateLimit struct {
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
