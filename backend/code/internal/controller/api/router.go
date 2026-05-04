package api

import (
	"net/http"

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
		}

		nodes := v1.Group("/nodes")
		{
			nodes.POST("/register", server.RegisterNode)
			nodes.POST("/:node_id/heartbeat", server.Heartbeat)
			nodes.GET("/:node_id/peers", server.GetPeers)
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
	respondOK(c, protocol.DashboardNodesResponse{
		Nodes: s.registry.DashboardNodes(),
	})
}

func (s *Server) RegisterNode(c *gin.Context) {
	var req protocol.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
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

	_, onlinePeerCount, ok := s.registry.Heartbeat(nodeID, req)
	if !ok {
		respondError(c, http.StatusNotFound, "node not found")
		return
	}

	action := protocol.ActionNone
	if onlinePeerCount > 0 {
		action = protocol.ActionSyncPeers
	}

	respondOK(c, protocol.HeartbeatResponse{
		ActionRequired: action,
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

func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, protocol.APIResponse[any]{
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
