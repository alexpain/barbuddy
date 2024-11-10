package config

import "github.com/caarlos0/env/v10"

type Config struct {
	App struct {
		Name     string `env:"APP_NAME" envDefault:"bar-buddy"`
		LogLevel string `env:"APP_LOG_LEVEL" envDefault:"info"`
	}
	Bot struct {
		Token string `env:"YOUR_TELEGRAM_BOT_TOKEN"`
	}
}

func New() (*Config, error) {
	cfg := new(Config)

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
