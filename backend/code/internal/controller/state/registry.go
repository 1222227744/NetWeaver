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

	"netweaver-backend/pkg/protocol"
)

const offlineAfter = time.Duration(protocol.DefaultKeepaliveInterval*3) * time.Second

type Registry struct {
	mu              sync.RWMutex
	nodes           map[string]protocol.NodeInfo
	machineIndex    map[string]string
	nextVirtualHost int
	startedAt       time.Time
}

func NewRegistry() *Registry {
	return &Registry{
		nodes:           make(map[string]protocol.NodeInfo),
		machineIndex:    make(map[string]string),
		nextVirtualHost: 2,
		startedAt:       time.Now(),
	}
}

func (r *Registry) Register(req protocol.RegisterNodeRequest, clientIP string) protocol.NodeInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	publicIP := strings.TrimSpace(clientIP)
	if publicIP == "" {
		publicIP = req.LocalIP
	}

	if existingID, ok := r.machineIndex[req.MachineID]; ok {
		node := r.nodes[existingID]
		node.MachineID = req.MachineID
		node.Hostname = req.Hostname
		node.OS = req.OS
		node.LocalIP = req.LocalIP
		node.LastSeen = now
		if node.PublicIP == "" {
			node.PublicIP = publicIP
		}
		if node.NATType == "" {
			node.NATType = protocol.DefaultNATType
		}
		r.nodes[existingID] = node
		return node
	}

	node := protocol.NodeInfo{
		NodeID:       r.nextNodeIDLocked(req.MachineID),
		MachineID:    req.MachineID,
		Hostname:     req.Hostname,
		OS:           req.OS,
		LocalIP:      req.LocalIP,
		VirtualIP:    r.nextVirtualIPLocked(),
		PublicIP:     publicIP,
		NATType:      protocol.DefaultNATType,
		RegisteredAt: now,
		LastSeen:     now,
	}

	r.nodes[node.NodeID] = node
	r.machineIndex[node.MachineID] = node.NodeID
	return node
}

func (r *Registry) Heartbeat(nodeID string, req protocol.HeartbeatRequest) (protocol.NodeInfo, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.nodes[nodeID]
	if !ok {
		return protocol.NodeInfo{}, 0, false
	}

	node.NATType = req.NATType
	node.PublicIP = req.PublicIP
	node.PublicPort = req.PublicPort
	node.CurrentRXBytes = req.CurrentRXBytes
	node.CurrentTXBytes = req.CurrentTXBytes
	node.LastSeen = time.Now()
	r.nodes[nodeID] = node

	return node, r.onlinePeerCountLocked(nodeID, node.LastSeen), true
}

func (r *Registry) DashboardStats() protocol.DashboardStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	stats := protocol.DashboardStats{
		TotalNodes:          len(r.nodes),
		ControllerUptimeSec: int64(now.Sub(r.startedAt).Seconds()),
	}

	var totalTrafficBytes uint64
	for _, node := range r.nodes {
		if r.isOnlineLocked(node, now) {
			stats.OnlineNodes++
		}
		totalTrafficBytes += node.CurrentRXBytes + node.CurrentTXBytes
	}
	stats.TotalTrafficGB = math.Round(float64(totalTrafficBytes)/(1024*1024*1024)*100) / 100

	return stats
}

func (r *Registry) DashboardNodes() []protocol.DashboardNode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	onlineCount := 0
	for _, node := range r.nodes {
		if r.isOnlineLocked(node, now) {
			onlineCount++
		}
	}

	nodes := make([]protocol.DashboardNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		status := protocol.NodeStatusOffline
		connectedPeers := 0
		if r.isOnlineLocked(node, now) {
			status = protocol.NodeStatusOnline
			connectedPeers = max(onlineCount-1, 0)
		}

		nodes = append(nodes, protocol.DashboardNode{
			NodeID:         node.NodeID,
			Hostname:       node.Hostname,
			VirtualIP:      node.VirtualIP,
			PublicIP:       node.PublicIP,
			NATType:        node.NATType,
			Status:         status,
			ConnectedPeers: connectedPeers,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes
}

func (r *Registry) PeersFor(nodeID string) ([]protocol.PeerInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	requester, ok := r.nodes[nodeID]
	if !ok {
		return nil, false
	}

	now := time.Now()
	peers := make([]protocol.PeerInfo, 0, len(r.nodes)-1)
	for _, node := range r.nodes {
		if node.NodeID == nodeID || !r.isOnlineLocked(node, now) {
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
	r.mu.RLock()
	defer r.mu.RUnlock()

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
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := make(map[string]protocol.NodeInfo, len(r.nodes))
	for id, node := range r.nodes {
		snapshot[id] = node
	}
	return snapshot
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
		candidate := fmt.Sprintf("10.0.%d.%d", hostIndex/253, hostIndex%253+2)
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

func (r *Registry) onlinePeerCountLocked(nodeID string, now time.Time) int {
	count := 0
	for _, node := range r.nodes {
		if node.NodeID != nodeID && r.isOnlineLocked(node, now) {
			count++
		}
	}
	return count
}

func (r *Registry) isOnlineLocked(node protocol.NodeInfo, now time.Time) bool {
	return !node.LastSeen.IsZero() && now.Sub(node.LastSeen) <= offlineAfter
}

func recommendMode(natTypes ...string) string {
	for _, natType := range natTypes {
		if strings.Contains(strings.ToLower(natType), "symmetric") {
			return protocol.RecommendModeRelay
		}
	}
	return protocol.RecommendModeP2P
}
