package protocol

import "time"

const (
	CodeSuccess = 200
	MsgSuccess  = "success"

	DefaultSubnetMask        = "255.255.255.0"
	DefaultKeepaliveInterval = 15
	DefaultNATType           = "Unknown"
	NodeStatusOnline         = "online"
	NodeStatusOffline        = "offline"
	ActionNone               = "none"
	ActionSyncPeers          = "sync_peers"
	RecommendModeP2P         = "p2p"
	RecommendModeRelay       = "relay"
)

type APIResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type PingResponse struct {
	Msg string `json:"msg"`
}

type RegisterNodeRequest struct {
	MachineID string `json:"machine_id" binding:"required"`
	Hostname  string `json:"hostname" binding:"required"`
	OS        string `json:"os" binding:"required"`
	LocalIP   string `json:"local_ip" binding:"required"`
}

type RegisterNodeResponse struct {
	NodeID            string `json:"node_id"`
	VirtualIP         string `json:"virtual_ip"`
	SubnetMask        string `json:"subnet_mask"`
	KeepaliveInterval int    `json:"keepalive_interval"`
}

type HeartbeatRequest struct {
	NATType        string `json:"nat_type" binding:"required"`
	PublicIP       string `json:"public_ip" binding:"required"`
	PublicPort     int    `json:"public_port" binding:"required"`
	CurrentRXBytes uint64 `json:"current_rx_bytes"`
	CurrentTXBytes uint64 `json:"current_tx_bytes"`
}

type HeartbeatResponse struct {
	ActionRequired string `json:"action_required"`
}

type DashboardStats struct {
	TotalNodes          int     `json:"total_nodes"`
	OnlineNodes         int     `json:"online_nodes"`
	TotalTrafficGB      float64 `json:"total_traffic_gb"`
	ControllerUptimeSec int64   `json:"controller_uptime_sec"`
}

type DashboardNode struct {
	NodeID         string `json:"node_id"`
	Hostname       string `json:"hostname"`
	VirtualIP      string `json:"virtual_ip"`
	PublicIP       string `json:"public_ip"`
	NATType        string `json:"nat_type"`
	Status         string `json:"status"`
	ConnectedPeers int    `json:"connected_peers"`
}

type DashboardNodesResponse struct {
	Nodes []DashboardNode `json:"nodes"`
}

type PeerInfo struct {
	TargetVirtualIP  string `json:"target_virtual_ip"`
	TargetPublicIP   string `json:"target_public_ip"`
	TargetPublicPort int    `json:"target_public_port"`
	NATType          string `json:"nat_type"`
	RecommendMode    string `json:"recommend_mode"`
}

type PeersResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type NodeInfo struct {
	NodeID         string    `json:"node_id"`
	MachineID      string    `json:"machine_id"`
	Hostname       string    `json:"hostname"`
	OS             string    `json:"os"`
	LocalIP        string    `json:"local_ip"`
	VirtualIP      string    `json:"virtual_ip"`
	PublicIP       string    `json:"public_ip"`
	PublicPort     int       `json:"public_port"`
	NATType        string    `json:"nat_type"`
	CurrentRXBytes uint64    `json:"current_rx_bytes"`
	CurrentTXBytes uint64    `json:"current_tx_bytes"`
	RegisteredAt   time.Time `json:"registered_at"`
	LastSeen       time.Time `json:"last_seen"`
}
