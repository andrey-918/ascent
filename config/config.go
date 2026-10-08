package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Env string

const (
	EnvTest Env = "test"
	EnvDev  Env = "dev"
)

type Config struct {
	DatabaseName     string `env:"DB_NAME"`
	DatabaseHost     string `env:"DB_HOST"`
	DatabasePort     string `env:"DB_PORT"`
	DatabasePortTest string `env:"DB_PORT_TEST"`
	DatabaseUser     string `env:"DB_USER"`
	DatabasePassword string `env:"DB_PASSWORD"`
	Env              Env    `env:"ENV" envDefault:"dev"`
}

func (c *Config) DatabaseUrl() string {
	port := c.DatabasePort
	if c.Env == EnvTest {
		port = c.DatabasePortTest
	}
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseHost,
		port,
		c.DatabaseName,
	)
}

func New() (*Config, error) {
	loadDotenv()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

func loadDotenv() {
	if os.Getenv("DB_HOST") != "" {
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for i := 0; i < 5; i++ {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
