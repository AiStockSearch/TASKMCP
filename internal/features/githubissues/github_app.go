package githubissues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GitHubApp struct {
	appID          int64
	installationID int64
	privateKey     any // *rsa.PrivateKey

	mu        sync.Mutex
	token     string
	tokenExp  time.Time
	configErr error
}

func NewGitHubAppFromEnv() *GitHubApp {
	a := &GitHubApp{}

	appIDStr := strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
	instIDStr := strings.TrimSpace(os.Getenv("GITHUB_APP_INSTALLATION_ID"))
	keyPEM := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PEM"))
	keyPath := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"))

	if appIDStr == "" || instIDStr == "" || (keyPEM == "" && keyPath == "") {
		a.configErr = errors.New("GitHub App not configured (need GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID, and GITHUB_APP_PRIVATE_KEY_PEM or GITHUB_APP_PRIVATE_KEY_PATH)")
		return a
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil || appID <= 0 {
		a.configErr = errors.New("invalid GITHUB_APP_ID")
		return a
	}
	instID, err := strconv.ParseInt(instIDStr, 10, 64)
	if err != nil || instID <= 0 {
		a.configErr = errors.New("invalid GITHUB_APP_INSTALLATION_ID")
		return a
	}

	var pemBytes []byte
	if keyPEM != "" {
		pemBytes = []byte(keyPEM)
	} else {
		b, err := os.ReadFile(filepath.Clean(keyPath))
		if err != nil {
			a.configErr = fmt.Errorf("read GITHUB_APP_PRIVATE_KEY_PATH: %w", err)
			return a
		}
		pemBytes = b
	}

	pk, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		a.configErr = fmt.Errorf("parse GitHub App private key PEM: %w", err)
		return a
	}

	a.appID = appID
	a.installationID = instID
	a.privateKey = pk
	return a
}

func (a *GitHubApp) ConfigError() error {
	if a == nil {
		return errors.New("GitHub App not initialized")
	}
	return a.configErr
}

func (a *GitHubApp) InstallationToken(ctx context.Context, httpClient *http.Client) (string, error) {
	if a == nil {
		return "", errors.New("GitHub App not initialized")
	}
	if a.configErr != nil {
		return "", a.configErr
	}

	a.mu.Lock()
	if a.token != "" && time.Until(a.tokenExp) > 2*time.Minute {
		t := a.token
		a.mu.Unlock()
		return t, nil
	}
	a.mu.Unlock()

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprintf("%d", a.appID),
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}
	j := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := j.SignedString(a.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign GitHub App JWT: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", a.installationID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+signed)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request installation token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("installation token request failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var payload struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if payload.Token == "" || payload.ExpiresAt == "" {
		return "", errors.New("invalid token response from GitHub")
	}

	exp, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("parse expires_at: %w", err)
	}

	a.mu.Lock()
	a.token = payload.Token
	a.tokenExp = exp.UTC()
	a.mu.Unlock()

	return payload.Token, nil
}

