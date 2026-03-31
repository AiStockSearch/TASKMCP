package env

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadDotEnv() {
	_ = godotenv.Load()
}

func Required(name string) (string, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", errors.New("missing " + name + " env var")
	}
	return v, nil
}

