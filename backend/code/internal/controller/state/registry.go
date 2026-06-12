package state

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/fnv"
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
	missedHeartbeatLimit      = 3
	metricRetentionWindow     = 24 * time.Hour
	rawMetricWindow           = time.Hour
	metricAggregationInterval = time.Minute
	pairKeySep                = "\x00"
)

type peerObservation struct {
	LastSeen time.Time
	Misses   int
}

type edgeTypeState struct {
	LinkType string
	LastSeen time.Time
}

type Registry struct {
	mu              sync.RWMutex
	nodes           map[string]protocol.NodeInfo
	machineIndex    map[string]string
	peerReports     map[string]map[string]peerObservation
	edgeTypes       map[string]edgeTypeState
	linkMetrics     map[string][]protocol.DashboardMetricPoint
	nextVirtualHost int
	startedAt       time.Time
}

func NewRegistry() *Registry {
	return &Registry{
		nodes:           make(map[string]protocol.NodeInfo),
		machineIndex:    make(map[string]string),
		peerReports:     make(map[string]map[string]peerObservation),
		edgeTypes:       make(map[string]edgeTypeState),
		linkMetrics:     make(map[string][]protocol.DashboardMetricPoint),
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
		delete(r.peerReports, existingID)
		r.removeEdgeTypesForNodeLocked(existingID)
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

func (r *Registry) Heartbeat(nodeID string, req protocol.HeartbeatRequest, _ string) (protocol.NodeInfo, int, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.refreshDerivedStateLocked(now)

	node, ok := r.nodes[nodeID]
	if !ok {
		return protocol.NodeInfo{}, 0, 0, false
	}

	natType := strings.TrimSpace(req.NATType)
	publicIP := strings.TrimSpace(req.PublicIP)
	node.NATType = natType
	node.PublicIP = publicIP
	node.PublicPort = req.PublicPort
	node.CurrentRXBytes = req.CurrentRXBytes
	node.CurrentTXBytes = req.CurrentTXBytes
	node.Status = protocol.NodeStatusOnline
	node.LastSeen = now
	r.nodes[nodeID] = node

	connectedTargets := r.applyPeerReportLocked(nodeID, req.ConnectedPeerIDs, now)
	r.appendLinkMetricsLocked(nodeID, connectedTargets, req.LinkMetrics, now)

	r.refreshDerivedStateLocked(now)
	node = r.nodes[nodeID]

	return node, node.ConnectedPeers, r.onlinePeerCountLocked(nodeID), true
}

func (r *Registry) DashboardStats() protocol.DashboardStatsData {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.refreshDerivedStateLocked(now)

	stats := protocol.DashboardStatsData{
		TotalNodes:          len(r.nodes),
		ControllerUptimeSec: int64(now.Sub(r.startedAt).Seconds()),
		LastUpdatedAt:       now.Unix(),
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

func (r *Registry) LinkMetrics(nodeID string, targetID string, window time.Duration) (protocol.DashboardNodeMetricsData, bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.nodes[nodeID]; !ok {
		return protocol.DashboardNodeMetricsData{}, false, false
	}
	if _, ok := r.nodes[targetID]; !ok {
		return protocol.DashboardNodeMetricsData{}, true, false
	}

	now := time.Now().UTC()
	points := filterMetricsByWindow(r.linkMetrics[directionKey(nodeID, targetID)], now, window)
	return protocol.DashboardNodeMetricsData{
		Metrics: aggregateMetrics(points, now, window),
	}, true, true
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

		mode := recommendModeFor(requester, node)
		peer := protocol.PeerInfo{
			TargetNodeID:     node.NodeID,
			TargetHostname:   node.Hostname,
			TargetVirtualIP:  node.VirtualIP,
			TargetPublicIP:   node.PublicIP,
			TargetPublicPort: node.PublicPort,
			NATType:          node.NATType,
			RecommendMode:    mode,
		}
		if mode == protocol.RecommendModeRelay {
			peer.RelayAddr = config.RelayAddr()
			peer.RelayPort = config.RelayPort()
			peer.RelaySessionID = relaySessionID(nodeID, node.NodeID)
		}
		peers = append(peers, peer)
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

func (r *Registry) applyPeerReportLocked(nodeID string, peerIDs []string, now time.Time) map[string]struct{} {
	validTargets := make(map[string]struct{}, len(peerIDs))
	for _, rawPeerID := range peerIDs {
		peerID := strings.TrimSpace(rawPeerID)
		if peerID == "" || peerID == nodeID {
			continue
		}
		if _, exists := r.nodes[peerID]; !exists {
			continue
		}
		validTargets[peerID] = struct{}{}
	}

	report := r.peerReports[nodeID]
	if report == nil {
		report = make(map[string]peerObservation)
		r.peerReports[nodeID] = report
	}

	for peerID, obs := range report {
		if _, stillConnected := validTargets[peerID]; stillConnected {
			obs.LastSeen = now
			obs.Misses = 0
			report[peerID] = obs
			continue
		}

		obs.Misses++
		if obs.Misses >= missedHeartbeatLimit {
			delete(report, peerID)
			continue
		}
		report[peerID] = obs
	}

	for peerID := range validTargets {
		if _, exists := report[peerID]; exists {
			continue
		}
		report[peerID] = peerObservation{
			LastSeen: now,
			Misses:   0,
		}
	}

	return validTargets
}

func (r *Registry) appendLinkMetricsLocked(nodeID string, connectedTargets map[string]struct{}, metrics []protocol.LinkMetricReport, now time.Time) {
	timestamp := now.Unix()
	for _, metric := range metrics {
		targetID := strings.TrimSpace(metric.TargetNodeID)
		if _, connected := connectedTargets[targetID]; !connected {
			continue
		}
		if _, exists := r.nodes[targetID]; !exists || targetID == nodeID {
			continue
		}

		linkType := normalizeLinkType(metric.LinkType)
		if linkType == "" {
			continue
		}

		key := directionKey(nodeID, targetID)
		points := appendOrReplaceMetricPoint(r.linkMetrics[key], protocol.DashboardMetricPoint{
			Timestamp: timestamp,
			LatencyMS: metric.LatencyMS,
		})
		r.linkMetrics[key] = trimMetricHistory(points, now)

		r.edgeTypes[edgeKey(nodeID, targetID)] = edgeTypeState{
			LinkType: linkType,
			LastSeen: now,
		}
	}
}

func (r *Registry) dashboardEdgesLocked() []protocol.DashboardEdge {
	onlineByID := r.onlineNodeSetLocked()
	edges := make([]protocol.DashboardEdge, 0, len(r.edgeTypes))

	for key, edgeType := range r.edgeTypes {
		source, target, ok := splitEdgeKey(key)
		if !ok || !onlineByID[source] || !onlineByID[target] {
			continue
		}
		if !r.hasMutualPeerReportLocked(source, target) {
			continue
		}
		if edgeType.LinkType == "" {
			continue
		}

		edges = append(edges, protocol.DashboardEdge{
			Source: source,
			Target: target,
			Type:   edgeType.LinkType,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})
	return edges
}

func (r *Registry) isOnlineLocked(node protocol.NodeInfo, now time.Time) bool {
	return !node.LastSeen.IsZero() && now.Sub(node.LastSeen) <= offlineAfter
}

func (r *Registry) refreshDerivedStateLocked(now time.Time) int {
	expiredCount := 0
	onlineByID := make(map[string]bool, len(r.nodes))

	for id, node := range r.nodes {
		if r.isOnlineLocked(node, now) {
			node.Status = protocol.NodeStatusOnline
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
		node.ConnectedPeers = r.connectedPeerCountLocked(id, onlineByID)
		r.nodes[id] = node
	}

	r.pruneInactiveEdgeTypesLocked(onlineByID)
	return expiredCount
}

func (r *Registry) connectedPeerCountLocked(nodeID string, onlineByID map[string]bool) int {
	count := 0
	for peerID := range r.peerReports[nodeID] {
		if !onlineByID[peerID] {
			continue
		}
		if r.hasMutualPeerReportLocked(nodeID, peerID) {
			count++
		}
	}
	return count
}

func (r *Registry) hasMutualPeerReportLocked(a string, b string) bool {
	if _, ok := r.peerReports[a][b]; !ok {
		return false
	}
	if _, ok := r.peerReports[b][a]; !ok {
		return false
	}
	return true
}

func (r *Registry) onlinePeerCountLocked(nodeID string) int {
	count := 0
	for id, node := range r.nodes {
		if id != nodeID && node.Status == protocol.NodeStatusOnline {
			count++
		}
	}
	return count
}

func (r *Registry) onlineNodeSetLocked() map[string]bool {
	onlineByID := make(map[string]bool, len(r.nodes))
	for id, node := range r.nodes {
		if node.Status == protocol.NodeStatusOnline {
			onlineByID[id] = true
		}
	}
	return onlineByID
}

func (r *Registry) pruneInactiveEdgeTypesLocked(onlineByID map[string]bool) {
	for key := range r.edgeTypes {
		source, target, ok := splitEdgeKey(key)
		if !ok || !onlineByID[source] || !onlineByID[target] || !r.hasMutualPeerReportLocked(source, target) {
			delete(r.edgeTypes, key)
		}
	}
}

func (r *Registry) removeEdgeTypesForNodeLocked(nodeID string) {
	for key := range r.edgeTypes {
		source, target, ok := splitEdgeKey(key)
		if !ok || source == nodeID || target == nodeID {
			delete(r.edgeTypes, key)
		}
	}
}

func dashboardNodeFromInfo(node protocol.NodeInfo) protocol.DashboardNode {
	lastSeen := int64(0)
	if !node.LastSeen.IsZero() {
		lastSeen = node.LastSeen.Unix()
	}

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
		LastSeen:       lastSeen,
	}
}

func recommendModeFor(nodes ...protocol.NodeInfo) string {
	for _, node := range nodes {
		if requiresRelay(node) {
			return protocol.RecommendModeRelay
		}
	}
	return protocol.RecommendModeP2P
}

func requiresRelay(node protocol.NodeInfo) bool {
	natType := strings.ToLower(strings.TrimSpace(node.NATType))
	if natType == strings.ToLower(protocol.DefaultNATType) {
		return true
	}
	if strings.Contains(natType, "symmetric") {
		return true
	}
	return strings.TrimSpace(node.PublicIP) == "" || node.PublicPort <= 0
}

func relaySessionID(a string, b string) uint32 {
	source, target := orderedPair(a, b)
	h := fnv.New32a()
	_, _ = h.Write([]byte(source))
	_, _ = h.Write([]byte(pairKeySep))
	_, _ = h.Write([]byte(target))
	sessionID := h.Sum32()
	if sessionID == 0 {
		return 1
	}
	return sessionID
}

func edgeKey(a string, b string) string {
	source, target := orderedPair(a, b)
	return source + pairKeySep + target
}

func directionKey(source string, target string) string {
	return source + pairKeySep + target
}

func splitEdgeKey(key string) (string, string, bool) {
	parts := strings.Split(key, pairKeySep)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func orderedPair(a string, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

func normalizeLinkType(linkType string) string {
	switch strings.ToLower(strings.TrimSpace(linkType)) {
	case protocol.LinkTypeP2P:
		return protocol.LinkTypeP2P
	case protocol.LinkTypeRelay:
		return protocol.LinkTypeRelay
	default:
		return ""
	}
}

func appendOrReplaceMetricPoint(points []protocol.DashboardMetricPoint, point protocol.DashboardMetricPoint) []protocol.DashboardMetricPoint {
	if len(points) > 0 && points[len(points)-1].Timestamp == point.Timestamp {
		points[len(points)-1] = point
		return points
	}
	return append(points, point)
}

func trimMetricHistory(points []protocol.DashboardMetricPoint, now time.Time) []protocol.DashboardMetricPoint {
	cutoff := now.Add(-metricRetentionWindow).Unix()
	first := 0
	for first < len(points) && points[first].Timestamp < cutoff {
		first++
	}
	if first == 0 {
		return points
	}
	trimmed := make([]protocol.DashboardMetricPoint, len(points)-first)
	copy(trimmed, points[first:])
	return trimmed
}

func filterMetricsByWindow(points []protocol.DashboardMetricPoint, now time.Time, window time.Duration) []protocol.DashboardMetricPoint {
	if window <= 0 {
		window = rawMetricWindow
	}
	cutoff := now.Add(-window).Unix()
	filtered := make([]protocol.DashboardMetricPoint, 0, len(points))
	for _, point := range points {
		if point.Timestamp >= cutoff {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

func aggregateMetrics(points []protocol.DashboardMetricPoint, now time.Time, window time.Duration) []protocol.DashboardMetricPoint {
	if window <= rawMetricWindow {
		copied := make([]protocol.DashboardMetricPoint, len(points))
		copy(copied, points)
		sortMetrics(copied)
		return copied
	}

	rawCutoff := now.Add(-rawMetricWindow).Unix()
	type bucketValue struct {
		sum   float64
		count int
	}
	buckets := make(map[int64]bucketValue)
	result := make([]protocol.DashboardMetricPoint, 0, len(points))

	for _, point := range points {
		if point.Timestamp >= rawCutoff {
			result = append(result, point)
			continue
		}

		bucket := point.Timestamp - point.Timestamp%int64(metricAggregationInterval.Seconds())
		value := buckets[bucket]
		value.sum += point.LatencyMS
		value.count++
		buckets[bucket] = value
	}

	bucketTimestamps := make([]int64, 0, len(buckets))
	for timestamp := range buckets {
		bucketTimestamps = append(bucketTimestamps, timestamp)
	}
	sort.Slice(bucketTimestamps, func(i, j int) bool {
		return bucketTimestamps[i] < bucketTimestamps[j]
	})

	for _, timestamp := range bucketTimestamps {
		value := buckets[timestamp]
		if value.count == 0 {
			continue
		}
		result = append(result, protocol.DashboardMetricPoint{
			Timestamp: timestamp,
			LatencyMS: math.Round(value.sum/float64(value.count)*100) / 100,
		})
	}

	sortMetrics(result)
	return result
}

func sortMetrics(points []protocol.DashboardMetricPoint) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})
}
