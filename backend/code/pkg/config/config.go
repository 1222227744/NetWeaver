package config

import "time"

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
)
