package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"
)

const (
	dashboardTokenTTLSeconds = int64(86400)
	defaultDashboardUsername = "admin"
	defaultDashboardPassword = "change-me"
	defaultJWTSecret         = "netweaver-dashboard-dev-secret"
)

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func validDashboardCredential(username string, password string) bool {
	return username == dashboardUsername() && password == dashboardPassword()
}

func issueDashboardToken(username string, now time.Time) (string, int64, error) {
	expiresAt := now.Unix() + dashboardTokenTTLSeconds
	header := jwtHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}
	claims := jwtClaims{
		Subject:   username,
		Issuer:    "netweaver-controller",
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt,
	}

	encodedHeader, err := encodeJWTPart(header)
	if err != nil {
		return "", 0, err
	}
	encodedClaims, err := encodeJWTPart(claims)
	if err != nil {
		return "", 0, err
	}

	signingInput := encodedHeader + "." + encodedClaims
	mac := hmac.New(sha256.New, []byte(jwtSecret()))
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, expiresAt, nil
}

func encodeJWTPart(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func dashboardUsername() string {
	return envOrDefault("NETWEAVER_DASHBOARD_USERNAME", defaultDashboardUsername)
}

func dashboardPassword() string {
	return envOrDefault("NETWEAVER_DASHBOARD_PASSWORD", defaultDashboardPassword)
}

func jwtSecret() string {
	return envOrDefault("NETWEAVER_JWT_SECRET", defaultJWTSecret)
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
