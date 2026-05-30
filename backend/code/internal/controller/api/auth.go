package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"netweaver-backend/pkg/config"
)

const (
	dashboardTokenTTLSeconds = int64(86400)
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
	return username == config.DashboardUsername() && password == config.DashboardPassword()
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
	signature := signJWT(signingInput)

	return signingInput + "." + signature, expiresAt, nil
}

func requireDashboardJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || !validDashboardToken(token, time.Now().UTC()) {
			respondError(c, 401, "unauthorized: invalid or expired token")
			c.Abort()
			return
		}

		c.Next()
	}
}

func requireNodePSK() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || !validNodePSK(token) {
			respondError(c, 401, "unauthorized: invalid or missing PSK")
			c.Abort()
			return
		}

		c.Next()
	}
}

func validDashboardToken(token string, now time.Time) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return false
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expectedSignature, err := base64.RawURLEncoding.DecodeString(signJWT(parts[0] + "." + parts[1]))
	if err != nil || !hmac.Equal(signature, expectedSignature) {
		return false
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return false
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return false
	}

	var claims jwtClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return false
	}
	if claims.Issuer != "netweaver-controller" || strings.TrimSpace(claims.Subject) == "" {
		return false
	}
	if claims.ExpiresAt <= now.Unix() {
		return false
	}
	if claims.IssuedAt > now.Add(5*time.Minute).Unix() {
		return false
	}

	return true
}

func validNodePSK(token string) bool {
	expected := config.NodePSK()
	return hmac.Equal([]byte(token), []byte(expected))
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func signJWT(signingInput string) string {
	mac := hmac.New(sha256.New, []byte(config.JWTSecret()))
	_, _ = mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeJWTPart(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeJWTPart(value string, out any) error {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
