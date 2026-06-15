package config

import (
	"os"
	"strconv"
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
	EnvRelayAddr         = "NETWEAVER_RELAY_ADDR"
	EnvRelayPort         = "NETWEAVER_RELAY_PORT"
	EnvSTUNServers       = "NETWEAVER_STUN_SERVERS"
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

func STUNServers() string {
	return EnvOrDefault(EnvSTUNServers, DefaultSTUNServers)
}

func RelayAddr() string {
	return EnvOrDefault(EnvRelayAddr, DefaultRelayAddr)
}

func RelayPort() int {
	value := strings.TrimSpace(os.Getenv(EnvRelayPort))
	if value == "" {
		return DefaultRelayPort
	}

	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 {
		return DefaultRelayPort
	}
	return port
}

func EnvOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func UsesDefaultDashboardCredential() bool {
	return DashboardUsername() == DefaultDashboardUsername && DashboardPassword() == DefaultDashboardPassword
}

func UsesDefaultJWTSecret() bool {
	return JWTSecret() == DefaultJWTSecret
}

func UsesDefaultNodePSK() bool {
	return NodePSK() == DefaultNodePSK
}

func UsesDefaultSTUNServers() bool {
	return STUNServers() == DefaultSTUNServers
}
