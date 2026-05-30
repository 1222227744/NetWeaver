package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"netweaver-backend/internal/controller/middleware"
	"netweaver-backend/internal/controller/state"
	"netweaver-backend/pkg/protocol"
)

type Server struct {
	registry *state.Registry
}

func NewRouter(registry *state.Registry) *gin.Engine {
	server := &Server{registry: registry}

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/", server.Ping)
	r.GET("/ping", server.Ping)
	r.GET("/nodes", server.ListNodes)

	v1 := r.Group("/api/v1")
	{
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/stats", server.GetDashboardStats)
			dashboard.GET("/nodes", server.GetDashboardNodes)
			dashboard.GET("/edges", server.GetDashboardEdges)
			dashboard.GET("/nodes/:node_id/metrics", server.GetDashboardNodeMetrics)
		}

		nodes := v1.Group("/nodes")
		{
			nodes.POST("/register", server.RegisterNode)
			nodes.POST("/:node_id/heartbeat", server.Heartbeat)
			nodes.GET("/:node_id/peers", server.GetPeers)
			nodes.GET("/:node_id/metrics", server.GetDashboardNodeMetrics)
		}
	}

	return r
}

func (s *Server) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, protocol.PingResponse{Msg: "pong"})
}

func (s *Server) ListNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"nodes": s.registry.List(),
	})
}

func (s *Server) GetDashboardStats(c *gin.Context) {
	respondOK(c, s.registry.DashboardStats())
}

func (s *Server) GetDashboardNodes(c *gin.Context) {
	respondOK(c, protocol.DashboardNodesData{
		Nodes: s.registry.DashboardNodes(),
	})
}

func (s *Server) GetDashboardEdges(c *gin.Context) {
	respondOK(c, protocol.DashboardEdgesData{
		Edges: s.registry.DashboardEdges(),
	})
}

func (s *Server) RegisterNode(c *gin.Context) {
	var req protocol.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.MachineID) == "" || strings.TrimSpace(req.Hostname) == "" || strings.TrimSpace(req.OS) == "" || strings.TrimSpace(req.LocalIP) == "" {
		respondError(c, http.StatusBadRequest, "machine_id, hostname, os and local_ip are required")
		return
	}

	node := s.registry.Register(req, c.ClientIP())
	respondOK(c, protocol.RegisterNodeResponse{
		NodeID:            node.NodeID,
		VirtualIP:         node.VirtualIP,
		SubnetMask:        protocol.DefaultSubnetMask,
		KeepaliveInterval: protocol.DefaultKeepaliveInterval,
	})
}

func (s *Server) Heartbeat(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node_id is required")
		return
	}

	var req protocol.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if msg := validateHeartbeat(req); msg != "" {
		respondError(c, http.StatusBadRequest, msg)
		return
	}

	_, connectedPeerCount, onlinePeerCount, ok := s.registry.Heartbeat(nodeID, req, c.ClientIP())
	if !ok {
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	action := protocol.ActionNone
	if onlinePeerCount > connectedPeerCount {
		action = protocol.ActionSyncPeers
	}

	respondOK(c, protocol.HeartbeatResponse{
		ActionRequired: action,
		ConnectedPeers: connectedPeerCount,
	})
}

func (s *Server) GetPeers(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node_id is required")
		return
	}

	peers, ok := s.registry.PeersFor(nodeID)
	if !ok {
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	respondOK(c, protocol.PeersResponse{
		Peers: peers,
	})
}

func (s *Server) GetDashboardNodeMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		respondError(c, http.StatusBadRequest, "node_id is required")
		return
	}

	targetID := strings.TrimSpace(c.Query("target_id"))
	if targetID == "" {
		respondError(c, http.StatusBadRequest, "target_id is required")
		return
	}
	if targetID == nodeID {
		respondError(c, http.StatusBadRequest, "target_id must be different from node_id")
		return
	}

	window, ok := parseTimeRange(c.DefaultQuery("time_range", "1h"))
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid time_range")
		return
	}

	metrics, nodeOK, targetOK := s.registry.LinkMetrics(nodeID, targetID, window)
	if !nodeOK {
		respondError(c, http.StatusNotFound, "node not found")
		return
	}
	if !targetOK {
		respondError(c, http.StatusNotFound, "target not found")
		return
	}

	respondOK(c, metrics)
}

func validateHeartbeat(req protocol.HeartbeatRequest) string {
	natType := strings.TrimSpace(req.NATType)
	if natType == "" {
		return "nat_type is required"
	}
	if req.ConnectedPeerIDs == nil {
		return "connected_peer_ids is required"
	}

	publicIP := strings.TrimSpace(req.PublicIP)
	unknownWithoutMapping := strings.EqualFold(natType, protocol.DefaultNATType) && publicIP == "" && req.PublicPort == 0
	if !unknownWithoutMapping {
		if publicIP == "" {
			return "public_ip is required"
		}
		if req.PublicPort <= 0 {
			return "public_port must be greater than 0"
		}
	}

	for _, metric := range req.LinkMetrics {
		if strings.TrimSpace(metric.TargetNodeID) == "" {
			return "link_metrics.target_node_id is required"
		}
		if metric.LatencyMS < 0 {
			return "link_metrics.latency_ms must be non-negative"
		}
		if !isValidLinkType(metric.LinkType) {
			return "link_metrics.link_type must be p2p or relay"
		}
	}

	return ""
}

func isValidLinkType(linkType string) bool {
	switch strings.TrimSpace(strings.ToLower(linkType)) {
	case protocol.LinkTypeP2P, protocol.LinkTypeRelay:
		return true
	default:
		return false
	}
}

func parseTimeRange(raw string) (time.Duration, bool) {
	switch raw {
	case "", "1h":
		return time.Hour, true
	case "12h":
		return 12 * time.Hour, true
	case "24h":
		return 24 * time.Hour, true
	default:
		return 0, false
	}
}

func respondOK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, protocol.APIResponse[T]{
		Code: protocol.CodeSuccess,
		Msg:  protocol.MsgSuccess,
		Data: data,
	})
}

func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, protocol.APIResponse[any]{
		Code: status,
		Msg:  msg,
		Data: nil,
	})
}
