package runtime

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

const (
	punchRetries     = 3
	punchTimeout     = 2 * time.Second
	probeTimeout     = 2 * time.Second
	probeMinInterval = 2 * time.Second
	probeMaxInterval = 10 * time.Second
	sessionIDPairSep = "\x00"
)

type peerLink struct {
	Peer      protocol.PeerInfo
	LinkType  string
	LatencyMS float64
	LastSeen  time.Time
}

type linkManager struct {
	agent *Agent

	mu              sync.Mutex
	peers           map[string]protocol.PeerInfo
	peerByVirtualIP map[string]string
	links           map[string]peerLink
	punchAckWaiters map[string]chan struct{}
	probeAckWaiters map[string]chan struct{}
}

func newLinkManager(agent *Agent) *linkManager {
	return &linkManager{
		agent:           agent,
		peers:           make(map[string]protocol.PeerInfo),
		peerByVirtualIP: make(map[string]string),
		links:           make(map[string]peerLink),
		punchAckWaiters: make(map[string]chan struct{}),
		probeAckWaiters: make(map[string]chan struct{}),
	}
}

func (m *linkManager) replacePeers(peers []protocol.PeerInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nextPeers := make(map[string]protocol.PeerInfo, len(peers))
	nextByVirtualIP := make(map[string]string, len(peers))
	for _, peer := range peers {
		nextPeers[peer.TargetNodeID] = peer
		nextByVirtualIP[peer.TargetVirtualIP] = peer.TargetNodeID
	}

	for nodeID := range m.links {
		if _, ok := nextPeers[nodeID]; !ok {
			delete(m.links, nodeID)
		}
	}

	m.peers = nextPeers
	m.peerByVirtualIP = nextByVirtualIP
}

func (m *linkManager) establishPeers(ctx context.Context, peers []protocol.PeerInfo) {
	for _, peer := range peers {
		if ctx.Err() != nil {
			return
		}

		if peer.RecommendMode == protocol.RecommendModeP2P {
			if latency, ok := m.tryP2P(ctx, peer); ok {
				m.setLink(peer, protocol.LinkTypeP2P, latency)
				continue
			}

			log.Printf("p2p punch failed: peer=%s, falling back to relay", peer.TargetNodeID)
		}

		m.seedRelay(peer)
		m.setLink(peer, protocol.LinkTypeRelay, 0)
	}
}

func (m *linkManager) snapshot() ([]string, []protocol.LinkMetricReport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	peerIDs := make([]string, 0, len(m.links))
	metrics := make([]protocol.LinkMetricReport, 0, len(m.links))
	for nodeID, link := range m.links {
		if now.Sub(link.LastSeen) > 3*m.agent.cfg.PeerInterval {
			delete(m.links, nodeID)
			continue
		}
		peerIDs = append(peerIDs, nodeID)
		metrics = append(metrics, protocol.LinkMetricReport{
			TargetNodeID: nodeID,
			LatencyMS:    link.LatencyMS,
			LinkType:     link.LinkType,
		})
	}

	sort.Strings(peerIDs)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].TargetNodeID < metrics[j].TargetNodeID
	})
	return peerIDs, metrics
}

func (m *linkManager) readLoop(ctx context.Context, conn *net.UDPConn) {
	buf := make([]byte, 64*1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("data socket read failed: %v", err)
				return
			}
		}

		packet, err := decodeNetWeaverPacket(buf[:n])
		if err != nil {
			continue
		}
		if packet.dstVirtualIP != m.agent.virtualIP && packet.dstVirtualIP != "0.0.0.0" {
			continue
		}

		peer, ok := m.peerBySourceVirtualIP(packet.srcVirtualIP)
		if !ok {
			continue
		}

		switch packet.messageType {
		case messageTypeData:
			m.ensureLink(peer, linkTypeFromPacket(m.agent.nodeID, peer, packet.sessionID))
			if err := m.agent.writeTUN(packet.payload); err != nil {
				log.Printf("write inbound data packet failed: peer=%s err=%v", peer.TargetNodeID, err)
			}
		case messageTypePunchProbe:
			if err := m.sendControl(peer, remoteAddr, messageTypePunchAck, packet.sessionID); err != nil {
				log.Printf("send punch ack failed: peer=%s err=%v", peer.TargetNodeID, err)
				continue
			}
			m.setLink(peer, protocol.LinkTypeP2P, 0)
		case messageTypePunchAck:
			m.setLink(peer, protocol.LinkTypeP2P, 0)
			m.notifyPunchAck(peer.TargetNodeID)
		case messageTypeKeepalive:
			m.ensureLink(peer, linkTypeFromPacket(m.agent.nodeID, peer, packet.sessionID))
			if err := m.sendControl(peer, remoteAddr, messageTypeKeepaliveAck, packet.sessionID); err != nil {
				log.Printf("send keepalive ack failed: peer=%s err=%v", peer.TargetNodeID, err)
			}
		case messageTypeKeepaliveAck:
			m.touchLink(peer.TargetNodeID)
			m.notifyProbeAck(peer.TargetNodeID)
		case messageTypeRelayConfirm:
			m.ensureLink(peer, protocol.LinkTypeRelay)
		}
	}
}

func (m *linkManager) probeLoop(ctx context.Context) {
	interval := probeInterval(m.agent.cfg.HeartbeatInterval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.probeLinks(ctx)
		}
	}
}

func (m *linkManager) probeLinks(ctx context.Context) {
	links := m.linkSnapshot()
	var wg sync.WaitGroup
	for _, link := range links {
		if ctx.Err() != nil {
			return
		}

		wg.Add(1)
		go func(link peerLink) {
			defer wg.Done()
			m.probeLink(ctx, link)
		}(link)
	}
	wg.Wait()
}

func (m *linkManager) probeLink(ctx context.Context, link peerLink) {
	remoteAddr, sessionID, ok := m.remoteAddrForLink(link)
	if !ok {
		return
	}

	ackCh := m.registerProbeWaiter(link.Peer.TargetNodeID)
	defer m.unregisterProbeWaiter(link.Peer.TargetNodeID)

	sentAt := time.Now()
	if err := m.sendControl(link.Peer, remoteAddr, messageTypeKeepalive, sessionID); err != nil {
		log.Printf("send keepalive probe failed: peer=%s err=%v", link.Peer.TargetNodeID, err)
		return
	}

	timer := time.NewTimer(probeTimeout)
	defer timer.Stop()

	select {
	case <-ackCh:
		m.recordProbeLatency(link.Peer.TargetNodeID, float64(time.Since(sentAt).Microseconds())/1000)
	case <-timer.C:
	case <-ctx.Done():
	}
}

func (m *linkManager) tryP2P(ctx context.Context, peer protocol.PeerInfo) (float64, bool) {
	if m.agent.dataConn == nil {
		return 0, false
	}
	if peer.TargetPublicIP == "" || peer.TargetPublicPort <= 0 {
		return 0, false
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", peer.TargetPublicIP, peer.TargetPublicPort))
	if err != nil {
		return 0, false
	}

	ackCh := m.registerPunchWaiter(peer.TargetNodeID)
	defer m.unregisterPunchWaiter(peer.TargetNodeID)

	for attempt := 0; attempt < punchRetries; attempt++ {
		sentAt := time.Now()
		if err := m.sendControl(peer, remoteAddr, messageTypePunchProbe, 0); err != nil {
			log.Printf("send punch probe failed: peer=%s err=%v", peer.TargetNodeID, err)
			return 0, false
		}

		select {
		case <-ackCh:
			return float64(time.Since(sentAt).Microseconds()) / 1000, true
		case <-time.After(punchTimeout):
		case <-ctx.Done():
			return 0, false
		}
	}

	return 0, false
}

func (m *linkManager) sendData(payload []byte) error {
	if m.agent.dataConn == nil {
		return fmt.Errorf("data socket is not open")
	}

	dstVirtualIP, ok := ipv4Dst(payload)
	if !ok {
		return fmt.Errorf("unsupported outbound packet: expected IPv4 packet")
	}
	if dstVirtualIP == m.agent.virtualIP {
		return fmt.Errorf("drop local outbound packet to self: %s", dstVirtualIP)
	}

	link, ok := m.linkForVirtualIP(dstVirtualIP)
	if !ok {
		return fmt.Errorf("no active link for destination virtual ip %s", dstVirtualIP)
	}

	remoteAddr, sessionID, ok := m.remoteAddrForLink(link)
	if !ok {
		return fmt.Errorf("no usable remote address for destination virtual ip %s", dstVirtualIP)
	}

	data, err := encodeNetWeaverPacket(messageTypeData, sessionID, m.agent.virtualIP, link.Peer.TargetVirtualIP, payload)
	if err != nil {
		return err
	}

	if _, err := m.agent.dataConn.WriteToUDP(data, remoteAddr); err != nil {
		return err
	}
	return nil
}

func (m *linkManager) sendControl(peer protocol.PeerInfo, remoteAddr *net.UDPAddr, messageType byte, sessionID uint32) error {
	if m.agent.dataConn == nil {
		return fmt.Errorf("data socket is not open")
	}

	data, err := encodeNetWeaverPacket(messageType, sessionID, m.agent.virtualIP, peer.TargetVirtualIP, nil)
	if err != nil {
		return err
	}

	_, err = m.agent.dataConn.WriteToUDP(data, remoteAddr)
	return err
}

func (m *linkManager) seedRelay(peer protocol.PeerInfo) {
	remoteAddr, sessionID, ok := relayRemoteAddr(m.agent.nodeID, peer)
	if !ok {
		return
	}

	if err := m.sendControl(peer, remoteAddr, messageTypeKeepalive, sessionID); err != nil {
		log.Printf("seed relay failed: peer=%s err=%v", peer.TargetNodeID, err)
	}
}

func (m *linkManager) remoteAddrForLink(link peerLink) (*net.UDPAddr, uint32, bool) {
	if link.LinkType == protocol.LinkTypeRelay {
		return relayRemoteAddr(m.agent.nodeID, link.Peer)
	}

	if link.Peer.TargetPublicIP == "" || link.Peer.TargetPublicPort <= 0 {
		return nil, 0, false
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", link.Peer.TargetPublicIP, link.Peer.TargetPublicPort))
	if err != nil {
		return nil, 0, false
	}
	return remoteAddr, 0, true
}

func (m *linkManager) setLink(peer protocol.PeerInfo, linkType string, latencyMS float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if latencyMS < 0 {
		latencyMS = 0
	}
	m.links[peer.TargetNodeID] = peerLink{
		Peer:      peer,
		LinkType:  linkType,
		LatencyMS: latencyMS,
		LastSeen:  time.Now().UTC(),
	}
}

func (m *linkManager) ensureLink(peer protocol.PeerInfo, linkType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, ok := m.links[peer.TargetNodeID]
	if !ok {
		m.links[peer.TargetNodeID] = peerLink{
			Peer:      peer,
			LinkType:  linkType,
			LatencyMS: 0,
			LastSeen:  time.Now().UTC(),
		}
		return
	}

	link.Peer = peer
	link.LinkType = linkType
	link.LastSeen = time.Now().UTC()
	m.links[peer.TargetNodeID] = link
}

func (m *linkManager) recordProbeLatency(nodeID string, latencyMS float64) {
	if latencyMS < 0 {
		latencyMS = 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	link, ok := m.links[nodeID]
	if !ok {
		return
	}
	link.LatencyMS = latencyMS
	link.LastSeen = time.Now().UTC()
	m.links[nodeID] = link
}

func (m *linkManager) touchLink(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, ok := m.links[nodeID]
	if !ok {
		return
	}
	link.LastSeen = time.Now().UTC()
	m.links[nodeID] = link
}

func (m *linkManager) peerBySourceVirtualIP(virtualIP string) (protocol.PeerInfo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodeID, ok := m.peerByVirtualIP[virtualIP]
	if !ok {
		return protocol.PeerInfo{}, false
	}
	peer, ok := m.peers[nodeID]
	return peer, ok
}

func (m *linkManager) linkForVirtualIP(virtualIP string) (peerLink, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodeID, ok := m.peerByVirtualIP[virtualIP]
	if !ok {
		return peerLink{}, false
	}

	link, ok := m.links[nodeID]
	if !ok {
		return peerLink{}, false
	}
	if time.Since(link.LastSeen) > 3*m.agent.cfg.PeerInterval {
		delete(m.links, nodeID)
		return peerLink{}, false
	}
	return link, true
}

func (m *linkManager) linkSnapshot() []peerLink {
	m.mu.Lock()
	defer m.mu.Unlock()

	links := make([]peerLink, 0, len(m.links))
	for _, link := range m.links {
		links = append(links, link)
	}
	return links
}

func (m *linkManager) registerPunchWaiter(nodeID string) <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan struct{}, 1)
	m.punchAckWaiters[nodeID] = ch
	return ch
}

func (m *linkManager) unregisterPunchWaiter(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.punchAckWaiters, nodeID)
}

func (m *linkManager) notifyPunchAck(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.punchAckWaiters[nodeID]
	if !ok {
		return
	}

	select {
	case ch <- struct{}{}:
	default:
	}
}

func (m *linkManager) registerProbeWaiter(nodeID string) <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan struct{}, 1)
	m.probeAckWaiters[nodeID] = ch
	return ch
}

func (m *linkManager) unregisterProbeWaiter(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.probeAckWaiters, nodeID)
}

func (m *linkManager) notifyProbeAck(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.probeAckWaiters[nodeID]
	if !ok {
		return
	}

	select {
	case ch <- struct{}{}:
	default:
	}
}

func relayRemoteAddr(localNodeID string, peer protocol.PeerInfo) (*net.UDPAddr, uint32, bool) {
	sessionID := peer.RelaySessionID
	if sessionID == 0 {
		sessionID = deterministicRelaySessionID(localNodeID, peer.TargetNodeID)
	}
	if sessionID == 0 {
		return nil, 0, false
	}

	addr := strings.TrimSpace(peer.RelayAddr)
	if addr == "" {
		addr = config.DefaultRelayAddr
	}
	port := peer.RelayPort
	if port <= 0 {
		port = config.DefaultRelayPort
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return nil, 0, false
	}
	return remoteAddr, sessionID, true
}

func linkTypeFromPacket(localNodeID string, peer protocol.PeerInfo, sessionID uint32) string {
	if sessionID != 0 && (peer.RelaySessionID == sessionID || deterministicRelaySessionID(localNodeID, peer.TargetNodeID) == sessionID) {
		return protocol.LinkTypeRelay
	}
	return protocol.LinkTypeP2P
}

func probeInterval(base time.Duration) time.Duration {
	if base <= 0 {
		return probeMaxInterval
	}

	interval := base / 2
	if interval < probeMinInterval {
		return probeMinInterval
	}
	if interval > probeMaxInterval {
		return probeMaxInterval
	}
	return interval
}

func deterministicRelaySessionID(a string, b string) uint32 {
	if a == "" || b == "" {
		return 0
	}
	if b < a {
		a, b = b, a
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(a))
	_, _ = h.Write([]byte(sessionIDPairSep))
	_, _ = h.Write([]byte(b))
	sessionID := h.Sum32()
	if sessionID == 0 {
		return 1
	}
	return sessionID
}

func ipv4Dst(packet []byte) (string, bool) {
	if len(packet) < 20 {
		return "", false
	}
	if packet[0]>>4 != 4 {
		return "", false
	}
	ihl := int(packet[0]&0x0f) * 4
	if ihl < 20 || len(packet) < ihl {
		return "", false
	}
	return net.IPv4(packet[16], packet[17], packet[18], packet[19]).String(), true
}
