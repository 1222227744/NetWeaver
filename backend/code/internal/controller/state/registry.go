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
	}

	var totalTrafficBytes uint64
	for _, node := range r.nodes {
		if node.Status == protocol.NodeStatusOnline {
			stats.OnlineNodes++
		}
		totalTrafficBytes += node.CurrentRXBytes + node.CurrentTXBytes
	}
	stats.TotalTrafficGB = math.Round(float64(totalTrafficBytes)/(1024*1024*1024)*100) / 100

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

func (r *Registry) LinkMetrics(nodeID string, targetID string, timeRange string) (protocol.LinkMetricsResponse, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	duration, err := parseMetricTimeRange(timeRange)
	if err != nil {
		return protocol.LinkMetricsResponse{}, true, err
	}

	now := time.Now().UTC()
	r.refreshDerivedStateLocked(now)

	source, sourceOK := r.nodes[nodeID]
	target, targetOK := r.nodes[targetID]
	if !sourceOK || !targetOK {
		return protocol.LinkMetricsResponse{}, false, nil
	}

	if source.Status != protocol.NodeStatusOnline || target.Status != protocol.NodeStatusOnline {
		return protocol.LinkMetricsResponse{Metrics: []protocol.LinkMetricPoint{}}, true, nil
	}

	start := now.Add(-duration)
	points := make([]protocol.LinkMetricPoint, 0, len(r.metrics[nodeID])+1)
	seenTimestamps := make(map[int64]bool, len(r.metrics[nodeID])+1)
	for _, point := range r.metrics[nodeID] {
		if point.Timestamp.Before(start) {
			continue
		}

		timestamp := point.Timestamp.Unix()
		if seenTimestamps[timestamp] {
			continue
		}
		seenTimestamps[timestamp] = true
		points = append(points, protocol.LinkMetricPoint{
			Timestamp: timestamp,
			LatencyMS: estimateLinkLatencyMS(source, target, point.Timestamp),
		})
	}

	if len(points) == 0 {
		points = append(points, protocol.LinkMetricPoint{
			Timestamp: now.Unix(),
			LatencyMS: estimateLinkLatencyMS(source, target, now),
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})

	return protocol.LinkMetricsResponse{Metrics: points}, true, nil
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
				Source: source.NodeID,
				Target: target.NodeID,
				Type:   recommendMode(source.NATType, target.NATType),
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
		Hostname:       node.Hostname,
		VirtualIP:      node.VirtualIP,
		PublicIP:       node.PublicIP,
		NATType:        node.NATType,
		Status:         node.Status,
		ConnectedPeers: node.ConnectedPeers,
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

func parseMetricTimeRange(raw string) (time.Duration, error) {
	timeRange := strings.TrimSpace(raw)
	if timeRange == "" {
		timeRange = "1h"
	}

	duration, err := time.ParseDuration(timeRange)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("time_range must be a positive duration such as 1h, 12h or 24h")
	}

	return duration, nil
}

func estimateLinkLatencyMS(source protocol.NodeInfo, target protocol.NodeInfo, timestamp time.Time) float64 {
	base := 28.0
	if recommendMode(source.NATType, target.NATType) == protocol.RecommendModeRelay {
		base = 78.0
	}

	if source.PublicIP == "" || target.PublicIP == "" {
		base += 12
	}

	jitter := float64((timestamp.Unix()/60+int64(len(source.NodeID)+len(target.NodeID)))%13) - 6
	latency := base + jitter
	if latency < 1 {
		latency = 1
	}

	return math.Round(latency*10) / 10
}
