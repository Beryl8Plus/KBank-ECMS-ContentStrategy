package model

type InboundConfig struct {
	Server []Server `yaml:"server"`
}

type Server struct {
	Name      string    `yaml:"name"`
	Endpoint  string    `yaml:"endpoint"`
	RateLimit RateLimit `yaml:"ratelimit"`
}

type RateLimit struct {
	RPS   int `yaml:"requests_per_second"`
	Burst int `yaml:"burst"`
	MCR   int `yaml:"max_concurrent_requests"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}
