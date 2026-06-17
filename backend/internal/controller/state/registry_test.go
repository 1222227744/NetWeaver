package state

import (
	"testing"

	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

func TestPeersForIncludesRelayFallbackForP2PRecommendation(t *testing.T) {
	t.Setenv(config.EnvRelayAddr, "203.0.113.10")
	t.Setenv(config.EnvRelayPort, "19000")

	registry := NewRegistry()
	nodeA := registry.Register(protocol.RegisterNodeRequest{
		MachineID: "machine-a",
		Hostname:  "node-a",
		OS:        "linux",
		LocalIP:   "192.168.1.10",
	}, "198.51.100.10")
	nodeB := registry.Register(protocol.RegisterNodeRequest{
		MachineID: "machine-b",
		Hostname:  "node-b",
		OS:        "linux",
		LocalIP:   "192.168.1.11",
	}, "198.51.100.11")

	registry.Heartbeat(nodeA.NodeID, protocol.HeartbeatRequest{
		NATType:          "Full Cone",
		PublicIP:         "198.51.100.10",
		PublicPort:       9101,
		ConnectedPeerIDs: []string{},
	}, "")
	registry.Heartbeat(nodeB.NodeID, protocol.HeartbeatRequest{
		NATType:          "Full Cone",
		PublicIP:         "198.51.100.11",
		PublicPort:       9102,
		ConnectedPeerIDs: []string{},
	}, "")

	peers, ok := registry.PeersFor(nodeA.NodeID)
	if !ok {
		t.Fatal("expected node-a peers to be available")
	}
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}

	peer := peers[0]
	if peer.TargetNodeID != nodeB.NodeID {
		t.Fatalf("expected node-b peer, got %q", peer.TargetNodeID)
	}
	if peer.RecommendMode != protocol.RecommendModeP2P {
		t.Fatalf("expected p2p recommendation, got %q", peer.RecommendMode)
	}
	if peer.RelayAddr != "203.0.113.10" {
		t.Fatalf("expected relay addr 203.0.113.10, got %q", peer.RelayAddr)
	}
	if peer.RelayPort != 19000 {
		t.Fatalf("expected relay port 19000, got %d", peer.RelayPort)
	}
	if peer.RelaySessionID == 0 {
		t.Fatal("expected relay session id for p2p fallback")
	}
}
