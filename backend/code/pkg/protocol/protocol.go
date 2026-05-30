package protocol

import "time"

const (
	CodeSuccess = 200
	MsgSuccess  = "success"

	DefaultSubnetMask        = "255.255.0.0"
	DefaultKeepaliveInterval = 15
	DefaultNATType           = "Unknown"
	NodeStatusOnline         = "online"
	NodeStatusOffline        = "offline"
	ActionNone               = "none"
	ActionSyncPeers          = "sync_peers"
	RecommendModeP2P         = "p2p"
	RecommendModeRelay       = "relay"
	LinkTypeP2P              = "p2p"
	LinkTypeRelay            = "relay"
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
	NATType          string             `json:"nat_type"`
	PublicIP         string             `json:"public_ip"`
	PublicPort       int                `json:"public_port"`
	CurrentRXBytes   uint64             `json:"current_rx_bytes"`
	CurrentTXBytes   uint64             `json:"current_tx_bytes"`
	ConnectedPeers   int                `json:"connected_peers,omitempty"`
	ConnectedPeerIDs []string           `json:"connected_peer_ids"`
	LinkMetrics      []LinkMetricReport `json:"link_metrics,omitempty"`
}

type LinkMetricReport struct {
	TargetNodeID string  `json:"target_node_id"`
	LatencyMS    float64 `json:"latency_ms"`
	LinkType     string  `json:"link_type"`
}

type HeartbeatResponse struct {
	ActionRequired string `json:"action_required"`
	ConnectedPeers int    `json:"connected_peers"`
}

type DashboardStatsData struct {
	TotalNodes          int     `json:"total_nodes"`
	OnlineNodes         int     `json:"online_nodes"`
	OfflineNodes        int     `json:"offline_nodes"`
	TotalEdges          int     `json:"total_edges"`
	TotalRXBytes        uint64  `json:"total_rx_bytes"`
	TotalTXBytes        uint64  `json:"total_tx_bytes"`
	TotalTrafficBytes   uint64  `json:"total_traffic_bytes"`
	TotalTrafficGB      float64 `json:"total_traffic_gb"`
	ControllerUptimeSec int64   `json:"controller_uptime_sec"`
	LastUpdatedAt       int64   `json:"last_updated_at"`
}

type DashboardNode struct {
	NodeID         string `json:"node_id"`
	MachineID      string `json:"machine_id"`
	Hostname       string `json:"hostname"`
	OS             string `json:"os"`
	LocalIP        string `json:"local_ip"`
	VirtualIP      string `json:"virtual_ip"`
	PublicIP       string `json:"public_ip"`
	PublicPort     int    `json:"public_port"`
	NATType        string `json:"nat_type"`
	Status         string `json:"status"`
	ConnectedPeers int    `json:"connected_peers"`
	CurrentRXBytes uint64 `json:"current_rx_bytes"`
	CurrentTXBytes uint64 `json:"current_tx_bytes"`
	LastSeen       int64  `json:"last_seen"`
}

type DashboardNodesData struct {
	Nodes []DashboardNode `json:"nodes"`
}

type DashboardEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type DashboardEdgesData struct {
	Edges []DashboardEdge `json:"edges"`
}

type PeerInfo struct {
	TargetNodeID     string `json:"target_node_id"`
	TargetHostname   string `json:"target_hostname"`
	TargetVirtualIP  string `json:"target_virtual_ip"`
	TargetPublicIP   string `json:"target_public_ip"`
	TargetPublicPort int    `json:"target_public_port"`
	NATType          string `json:"nat_type"`
	RecommendMode    string `json:"recommend_mode"`
	RelayAddr        string `json:"relay_addr,omitempty"`
	RelayPort        int    `json:"relay_port,omitempty"`
	RelaySessionID   uint32 `json:"relay_session_id,omitempty"`
}

type PeersResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type DashboardMetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	LatencyMS float64 `json:"latency_ms"`
}

type DashboardNodeMetricsData struct {
	Metrics []DashboardMetricPoint `json:"metrics"`
}

type NodeMetricsResponse = DashboardNodeMetricsData

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
	Status         string    `json:"status"`
	ConnectedPeers int       `json:"connected_peers"`
	CurrentRXBytes uint64    `json:"current_rx_bytes"`
	CurrentTXBytes uint64    `json:"current_tx_bytes"`
	RegisteredAt   time.Time `json:"registered_at"`
	LastSeen       time.Time `json:"last_seen"`
}

type DashboardStats = DashboardStatsData
type DashboardNodesResponse = DashboardNodesData
