// Package twelvedata provides a client for the Twelve Data stock market API.
package twelvedata

import (
	"os"
	"time"
)

// Config holds configuration for the Twelve Data API client.
type Config struct {
	TwelveDataAPIKey string        // API key for authentication
	BaseURL          string        // Base URL for the API (e.g., "https://api.twelvedata.com")
	Timeout          time.Duration // HTTP request timeout
}

// LoadConfig loads Twelve Data configuration from environment variables.
func LoadConfig() Config {
	return Config{
		TwelveDataAPIKey: os.Getenv("TWELVE_DATA_API_KEY"),
		BaseURL:          os.Getenv("TWELVE_DATA_BASE_URL"),
		Timeout:          10 * time.Second,
	}
}
