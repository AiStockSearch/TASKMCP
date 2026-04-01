package env

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads `.env` from the current working directory, then — if DATABASE_URL
// is still empty — from the first `.env` found walking up parent directories.
// Cursor often starts the MCP subprocess with cwd outside the repo; without this,
// DATABASE_URL is missing and MCP may inherit a wrong global env.
func LoadDotEnv() {
	_ = godotenv.Load()
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
		return
	}
	if p := strings.TrimSpace(os.Getenv("MCP_VAULT_BRIDGE_DOTENV")); p != "" {
		_ = godotenv.Load(p)
		if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
			return
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for dir := wd; dir != ""; {
			p := filepath.Join(dir, ".env")
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				_ = godotenv.Load(p)
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
}

func Required(name string) (string, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", errors.New("missing " + name + " env var")
	}
	return v, nil
}

