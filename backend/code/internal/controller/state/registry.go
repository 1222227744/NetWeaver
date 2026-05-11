package state

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

const (
	offlineAfter              = config.DefaultHeartbeatInterval * 3
	defaultMetricHistoryLimit = 120
)

type Registry struct {
	mu              sync.RWMutex
	nodes           map[string]protocol.NodeInfo
	machineIndex    map[string]string
	metrics         map[string][]protocol.NodeMetricPoint
	nextVirtualHost int
	startedAt       time.Time
}

func NewRegistry() *Registry {
	return &Registry{
		nodes:           make(map[string]protocol.NodeInfo),
		machineIndex:    make(map[string]string),
		metrics:         make(map[string][]protocol.NodeMetricPoint),
		nextVirtualHost: 2,
		startedAt:       time.Now().UTC(),
	}
}

func (r *Registry) Register(req protocol.RegisterNodeRequest, clientIP string) protocol.NodeInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	machineID := strings.TrimSpace(req.MachineID)
	hostname := strings.TrimSpace(req.Hostname)
	osName := strings.TrimSpace(req.OS)
	localIP := strings.TrimSpace(req.LocalIP)
	publicIP := strings.TrimSpace(clientIP)
	if publicIP == "" {
		publicIP = localIP
	}

	if existingID, ok := r.machineIndex[machineID]; ok {
		node := r.nodes[existingID]
		node.MachineID = machineID
		node.Hostname = hostname
		node.OS = osName
		node.LocalIP = localIP
		node.LastSeen = now
		node.Status = protocol.NodeStatusOnline
		if node.PublicIP == "" {
			node.PublicIP = publicIP
		}
		if node.NATType == "" {
			node.NATType = protocol.DefaultNATType
		}
		r.nodes[existingID] = node
		r.refreshDerivedStateLocked(now)
		return node
	}

	node := protocol.NodeInfo{
		NodeID:       r.nextNodeIDLocked(machineID),
		MachineID:    machineID,
		Hostname:     hostname,
		OS:           osName,
		LocalIP:      localIP,
		VirtualIP:    r.nextVirtualIPLocked(),
		PublicIP:     publicIP,
		NATType:      protocol.DefaultNATType,
		Status:       protocol.NodeStatusOnline,
		RegisteredAt: now,
		LastSeen:     now,
	}

	r.nodes[node.NodeID] = node
	r.machineIndex[node.MachineID] = node.NodeID
	r.refreshDerivedStateLocked(now)
	return node
}

func (r *Registry) Heartbeat(nodeID string, req protocol.HeartbeatRequest, clientIP string) (protocol.NodeInfo, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.nodes[nodeID]
	if !ok {
		return protocol.NodeInfo{}, 0, false
	}

	natType := strings.TrimSpace(req.NATType)
	if natType == "" {
		natType = protocol.DefaultNATType
	}

	publicIP := strings.TrimSpace(req.PublicIP)
	if publicIP == "" {
		publicIP = strings.TrimSpace(clientIP)
	}
	if publicIP == "" {
		publicIP = node.PublicIP
	}

	node.NATType = natType
	node.PublicIP = publicIP
	node.PublicPort = req.PublicPort
	node.CurrentRXBytes = req.CurrentRXBytes
	node.CurrentTXBytes = req.CurrentTXBytes
	node.Status = protocol.NodeStatusOnline
	node.LastSeen = time.Now().UTC()
	r.nodes[nodeID] = node

	r.refreshDerivedStateLocked(node.LastSeen)
	node = r.nodes[nodeID]
	r.appendMetricLocked(node)

	return node, node.ConnectedPeers, true
}

func (r *Registry) DashboardStats() protocol.DashboardStatsData {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.refreshDerivedStateLocked(now)

	stats := protocol.DashboardStatsData{
		TotalNodes:          len(r.nodes),
		ControllerUptimeSec: int64(now.Sub(r.startedAt).Seconds()),
		LastUpdatedAt:       now,
	}

	for _, node := range r.nodes {
		if node.Status == protocol.NodeStatusOnline {
			stats.OnlineNodes++
		}
		stats.TotalRXBytes += node.CurrentRXBytes
		stats.TotalTXBytes += node.CurrentTXBytes
	}
	stats.OfflineNodes = stats.TotalNodes - stats.OnlineNodes
	stats.TotalEdges = len(r.dashboardEdgesLocked())
	stats.TotalTrafficBytes = stats.TotalRXBytes + stats.TotalTXBytes
	stats.TotalTrafficGB = math.Round(float64(stats.TotalTrafficBytes)/(1024*1024*1024)*100) / 100

	return stats
}

func (r *Registry) DashboardNodes() []protocol.DashboardNode {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshDerivedStateLocked(time.Now().UTC())
	nodes := make([]protocol.DashboardNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, dashboardNodeFromInfo(node))
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes
}

func (r *Registry) DashboardEdges() []protocol.DashboardEdge {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshDerivedStateLocked(time.Now().UTC())
	return r.dashboardEdgesLocked()
}

func (r *Registry) PeersFor(nodeID string) ([]protocol.PeerInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	requester, ok := r.nodes[nodeID]
	if !ok {
		return nil, false
	}

	r.refreshDerivedStateLocked(time.Now().UTC())
	requester = r.nodes[nodeID]
	if requester.Status != protocol.NodeStatusOnline {
		return []protocol.PeerInfo{}, true
	}

	peers := make([]protocol.PeerInfo, 0, len(r.nodes)-1)
	for _, node := range r.nodes {
		if node.NodeID == nodeID || node.Status != protocol.NodeStatusOnline {
			continue
		}

		peers = append(peers, protocol.PeerInfo{
			TargetNodeID:     node.NodeID,
			TargetHostname:   node.Hostname,
			TargetVirtualIP:  node.VirtualIP,
			TargetPublicIP:   node.PublicIP,
			TargetPublicPort: node.PublicPort,
			NATType:          node.NATType,
			RecommendMode:    recommendMode(requester.NATType, node.NATType),
		})
	}

	sort.Slice(peers, func(i, j int) bool {
		return peers[i].TargetVirtualIP < peers[j].TargetVirtualIP
	})
	return peers, true
}

func (r *Registry) List() []protocol.NodeInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshDerivedStateLocked(time.Now().UTC())
	nodes := make([]protocol.NodeInfo, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes
}

func (r *Registry) Snapshot() map[string]protocol.NodeInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshDerivedStateLocked(time.Now().UTC())
	snapshot := make(map[string]protocol.NodeInfo, len(r.nodes))
	for id, node := range r.nodes {
		snapshot[id] = node
	}
	return snapshot
}

func (r *Registry) CleanupExpiredNodes() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.refreshDerivedStateLocked(time.Now().UTC())
}

func (r *Registry) NodeMetrics(nodeID string, limit int) (protocol.NodeMetricsResponse, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.nodes[nodeID]; !ok {
		return protocol.NodeMetricsResponse{}, false
	}

	points := r.metrics[nodeID]
	if limit <= 0 || limit > len(points) {
		limit = len(points)
	}

	start := len(points) - limit
	if start < 0 {
		start = 0
	}

	copied := make([]protocol.NodeMetricPoint, len(points[start:]))
	copy(copied, points[start:])

	return protocol.NodeMetricsResponse{
		NodeID: nodeID,
		Points: copied,
	}, true
}

func (r *Registry) nextNodeIDLocked(machineID string) string {
	sum := sha1.Sum([]byte(machineID))
	base := "nw-node-" + hex.EncodeToString(sum[:])[:8]
	candidate := base
	for i := 2; ; i++ {
		if _, exists := r.nodes[candidate]; !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func (r *Registry) nextVirtualIPLocked() string {
	for {
		hostIndex := r.nextVirtualHost - 2
		candidate := fmt.Sprintf("%s.%d.%d", config.DefaultVirtualIPPrefix, hostIndex/253, hostIndex%253+2)
		r.nextVirtualHost++
		if !r.virtualIPInUseLocked(candidate) {
			return candidate
		}
	}
}

func (r *Registry) virtualIPInUseLocked(virtualIP string) bool {
	for _, node := range r.nodes {
		if node.VirtualIP == virtualIP {
			return true
		}
	}
	return false
}

func (r *Registry) dashboardEdgesLocked() []protocol.DashboardEdge {
	onlineNodes := make([]protocol.NodeInfo, 0, len(r.nodes))
	for _, node := range r.nodes {
		if node.Status == protocol.NodeStatusOnline {
			onlineNodes = append(onlineNodes, node)
		}
	}

	sort.Slice(onlineNodes, func(i, j int) bool {
		return onlineNodes[i].NodeID < onlineNodes[j].NodeID
	})

	edges := make([]protocol.DashboardEdge, 0, len(onlineNodes))
	for i := 0; i < len(onlineNodes); i++ {
		for j := i + 1; j < len(onlineNodes); j++ {
			source := onlineNodes[i]
			target := onlineNodes[j]
			edges = append(edges, protocol.DashboardEdge{
				EdgeID:          source.NodeID + "__" + target.NodeID,
				SourceNodeID:    source.NodeID,
				TargetNodeID:    target.NodeID,
				SourceHostname:  source.Hostname,
				TargetHostname:  target.Hostname,
				SourceVirtualIP: source.VirtualIP,
				TargetVirtualIP: target.VirtualIP,
				SourcePublicIP:  source.PublicIP,
				TargetPublicIP:  target.PublicIP,
				RecommendMode:   recommendMode(source.NATType, target.NATType),
				Status:          protocol.EdgeStatusActive,
				LastSeen:        olderTime(source.LastSeen, target.LastSeen),
			})
		}
	}

	return edges
}

func (r *Registry) isOnlineLocked(node protocol.NodeInfo, now time.Time) bool {
	return !node.LastSeen.IsZero() && now.Sub(node.LastSeen) <= offlineAfter
}

func (r *Registry) refreshDerivedStateLocked(now time.Time) int {
	onlineCount := 0
	expiredCount := 0
	onlineByID := make(map[string]bool, len(r.nodes))

	for id, node := range r.nodes {
		if r.isOnlineLocked(node, now) {
			node.Status = protocol.NodeStatusOnline
			onlineCount++
			onlineByID[id] = true
		} else {
			if node.Status != protocol.NodeStatusOffline {
				expiredCount++
			}
			node.Status = protocol.NodeStatusOffline
			node.ConnectedPeers = 0
		}
		r.nodes[id] = node
	}

	for id, node := range r.nodes {
		if !onlineByID[id] {
			continue
		}
		node.ConnectedPeers = max(onlineCount-1, 0)
		r.nodes[id] = node
	}

	return expiredCount
}

func (r *Registry) appendMetricLocked(node protocol.NodeInfo) {
	points := append(r.metrics[node.NodeID], protocol.NodeMetricPoint{
		Timestamp:      node.LastSeen,
		CurrentRXBytes: node.CurrentRXBytes,
		CurrentTXBytes: node.CurrentTXBytes,
		ConnectedPeers: node.ConnectedPeers,
		Status:         node.Status,
	})

	if len(points) > defaultMetricHistoryLimit {
		points = points[len(points)-defaultMetricHistoryLimit:]
	}

	r.metrics[node.NodeID] = points
}

func dashboardNodeFromInfo(node protocol.NodeInfo) protocol.DashboardNode {
	return protocol.DashboardNode{
		NodeID:         node.NodeID,
		MachineID:      node.MachineID,
		Hostname:       node.Hostname,
		OS:             node.OS,
		LocalIP:        node.LocalIP,
		VirtualIP:      node.VirtualIP,
		PublicIP:       node.PublicIP,
		PublicPort:     node.PublicPort,
		NATType:        node.NATType,
		Status:         node.Status,
		ConnectedPeers: node.ConnectedPeers,
		CurrentRXBytes: node.CurrentRXBytes,
		CurrentTXBytes: node.CurrentTXBytes,
		LastSeen:       node.LastSeen,
	}
}

func olderTime(a time.Time, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	if a.Before(b) {
		return a
	}
	return b
}

func recommendMode(natTypes ...string) string {
	for _, natType := range natTypes {
		if strings.Contains(strings.ToLower(natType), "symmetric") {
			return protocol.RecommendModeRelay
		}
	}
	return protocol.RecommendModeP2P
}
