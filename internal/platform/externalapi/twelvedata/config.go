package twelvedata

import (
	"os"
	"time"
)

type Config struct {
	TwelveDataAPIKey string
	BaseURL          string
	Timeout          time.Duration
}

func LoadConfig() Config {
	return Config{
		TwelveDataAPIKey: os.Getenv("TWELVE_DATA_API_KEY"),
		BaseURL:          os.Getenv("TWELVE_DATA_BASE_URL"),
		Timeout:          10 * time.Second,
	}
}
