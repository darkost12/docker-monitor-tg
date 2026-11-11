package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Token         string
	AllowedUserID int64
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("reading .env: %w", err)
	}
	return Parse(os.Getenv)
}

func Parse(getenv func(string) string) (Config, error) {
	token := getenv("TELEGRAM_BOT_TOKEN")
	idStr := getenv("ALLOWED_USER_ID")
	if token == "" || idStr == "" {
		return Config{}, errors.New("need to set TELEGRAM_BOT_TOKEN and ALLOWED_USER_ID (in .env or environment variables)")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("ALLOWED_USER_ID should be a number: %w", err)
	}
	return Config{Token: token, AllowedUserID: id}, nil
}
