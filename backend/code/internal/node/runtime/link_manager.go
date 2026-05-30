package runtime

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"netweaver-backend/pkg/protocol"
)

const (
	punchRetries = 3
	punchTimeout = 2 * time.Second
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
}

func newLinkManager(agent *Agent) *linkManager {
	return &linkManager{
		agent:           agent,
		peers:           make(map[string]protocol.PeerInfo),
		peerByVirtualIP: make(map[string]string),
		links:           make(map[string]peerLink),
		punchAckWaiters: make(map[string]chan struct{}),
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
			if err := m.sendControl(peer, remoteAddr, messageTypeKeepaliveAck, packet.sessionID); err != nil {
				log.Printf("send keepalive ack failed: peer=%s err=%v", peer.TargetNodeID, err)
			}
		case messageTypeKeepaliveAck:
			m.touchLink(peer.TargetNodeID)
		}
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
