package twelvedata

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TwelveDataAPIKey string
	BaseURL          string
	Timeout          time.Duration
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env ファイルが見つかりませんでした")
	}

	return Config{
		TwelveDataAPIKey: os.Getenv("TWELVE_DATA_API_KEY"),
		BaseURL:          os.Getenv("TWELVE_DATA_BASE_URL"),
		Timeout:          10 * time.Second,
	}
}
