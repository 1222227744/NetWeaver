package api

import (
	"net/http"
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
	r.POST("/nodes", server.RegisterNode)

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

func (s *Server) RegisterNode(c *gin.Context) {
	var req protocol.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if req.Address == "" {
		req.Address = c.ClientIP()
	}

	node := s.registry.Upsert(protocol.NodeInfo{
		ID:       req.ID,
		Address:  req.Address,
		LastSeen: time.Now(),
	})

	c.JSON(http.StatusOK, node)
}
