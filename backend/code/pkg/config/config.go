package config

import (
	"os"
	"strings"
	"time"
)

const (
	DefaultControllerAddr    = ":8080"
	DefaultControllerURL     = "http://127.0.0.1:8080"
	DefaultRequestTimeout    = 5 * time.Second
	DefaultHeartbeatInterval = 15 * time.Second
	DefaultVirtualSubnetCIDR = "10.0.0.0/16"
	DefaultVirtualIPPrefix   = "10.0"
	DefaultTUNName           = "tuno"
	DefaultRelayAddr         = "127.0.0.1"
	DefaultRelayPort         = 9000
	DefaultSTUNServers       = "stun.l.google.com:19302,stun1.l.google.com:19302,stun2.l.google.com:19302"
	DefaultP2PMessage        = "Hello"
	DefaultNodeAAddr         = "0.0.0.0:9001"
	DefaultNodeBAddr         = "0.0.0.0:9002"
	DefaultDashboardUsername = "admin"
	DefaultDashboardPassword = "change-me"
	DefaultJWTSecret         = "netweaver-dashboard-dev-secret"
	DefaultNodePSK           = "netweaver-dev-psk"

	EnvDashboardUsername = "NETWEAVER_DASHBOARD_USERNAME"
	EnvDashboardPassword = "NETWEAVER_DASHBOARD_PASSWORD"
	EnvJWTSecret         = "NETWEAVER_JWT_SECRET"
	EnvNodePSK           = "NETWEAVER_NODE_PSK"
)

func DashboardUsername() string {
	return EnvOrDefault(EnvDashboardUsername, DefaultDashboardUsername)
}

func DashboardPassword() string {
	return EnvOrDefault(EnvDashboardPassword, DefaultDashboardPassword)
}

func JWTSecret() string {
	return EnvOrDefault(EnvJWTSecret, DefaultJWTSecret)
}

func NodePSK() string {
	return EnvOrDefault(EnvNodePSK, DefaultNodePSK)
}

func EnvOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
