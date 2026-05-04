package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"netweaver-backend/internal/controller/state"
	"netweaver-backend/pkg/protocol"
)

type responseEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestRegisterHeartbeatAndDashboardEndpoints(t *testing.T) {
	router := newTestRouter()

	register := registerNode(t, router, "e4:5f:01:aa:bb:cc", "ubuntu-server-01", "linux", "192.168.1.10")
	if register.NodeID == "" {
		t.Fatal("expected generated node id")
	}
	if register.VirtualIP != "10.0.0.2" {
		t.Fatalf("expected first virtual ip 10.0.0.2, got %q", register.VirtualIP)
	}
	if register.SubnetMask != protocol.DefaultSubnetMask {
		t.Fatalf("expected subnet mask %q, got %q", protocol.DefaultSubnetMask, register.SubnetMask)
	}
	if register.KeepaliveInterval != protocol.DefaultKeepaliveInterval {
		t.Fatalf("expected keepalive interval %d, got %d", protocol.DefaultKeepaliveInterval, register.KeepaliveInterval)
	}

	heartbeatBody := `{
		"nat_type":"Full Cone",
		"public_ip":"203.0.113.5",
		"public_port":54321,
		"current_rx_bytes":1024560,
		"current_tx_bytes":2048000
	}`
	heartbeatResp := performJSON(t, router, http.MethodPost, "/api/v1/nodes/"+register.NodeID+"/heartbeat", heartbeatBody)
	if heartbeatResp.Code != http.StatusOK {
		t.Fatalf("expected heartbeat HTTP 200, got %d: %s", heartbeatResp.Code, heartbeatResp.Body.String())
	}
	heartbeat := decodeResponseData[protocol.HeartbeatResponse](t, heartbeatResp)
	if heartbeat.ActionRequired != protocol.ActionNone {
		t.Fatalf("expected no action for single-node heartbeat, got %q", heartbeat.ActionRequired)
	}

	nodesResp := performJSON(t, router, http.MethodGet, "/api/v1/dashboard/nodes", "")
	if nodesResp.Code != http.StatusOK {
		t.Fatalf("expected dashboard nodes HTTP 200, got %d: %s", nodesResp.Code, nodesResp.Body.String())
	}
	nodes := decodeResponseData[protocol.DashboardNodesResponse](t, nodesResp)
	if len(nodes.Nodes) != 1 {
		t.Fatalf("expected one dashboard node, got %d", len(nodes.Nodes))
	}
	node := nodes.Nodes[0]
	if node.NodeID != register.NodeID || node.Hostname != "ubuntu-server-01" || node.VirtualIP != "10.0.0.2" {
		t.Fatalf("dashboard node did not match registered node: %+v", node)
	}
	if node.NATType != "Full Cone" || node.PublicIP != "203.0.113.5" || node.Status != protocol.NodeStatusOnline {
		t.Fatalf("dashboard node did not reflect heartbeat state: %+v", node)
	}
	if node.ConnectedPeers != 0 {
		t.Fatalf("expected no connected peers for single online node, got %d", node.ConnectedPeers)
	}

	statsResp := performJSON(t, router, http.MethodGet, "/api/v1/dashboard/stats", "")
	if statsResp.Code != http.StatusOK {
		t.Fatalf("expected dashboard stats HTTP 200, got %d: %s", statsResp.Code, statsResp.Body.String())
	}
	stats := decodeResponseData[protocol.DashboardStats](t, statsResp)
	if stats.TotalNodes != 1 || stats.OnlineNodes != 1 {
		t.Fatalf("expected one total/online node, got %+v", stats)
	}
	if stats.ControllerUptimeSec < 0 {
		t.Fatalf("expected non-negative controller uptime, got %d", stats.ControllerUptimeSec)
	}
}

func TestPeersEndpointRecommendsRelayForSymmetricNAT(t *testing.T) {
	router := newTestRouter()

	nodeA := registerNode(t, router, "machine-a", "node-a", "linux", "192.168.1.10")
	nodeB := registerNode(t, router, "machine-b", "node-b", "darwin", "192.168.1.11")

	performJSON(t, router, http.MethodPost, "/api/v1/nodes/"+nodeA.NodeID+"/heartbeat", `{
		"nat_type":"Full Cone",
		"public_ip":"203.0.113.5",
		"public_port":54321,
		"current_rx_bytes":1,
		"current_tx_bytes":2
	}`)
	performJSON(t, router, http.MethodPost, "/api/v1/nodes/"+nodeB.NodeID+"/heartbeat", `{
		"nat_type":"Symmetric",
		"public_ip":"198.51.100.12",
		"public_port":45678,
		"current_rx_bytes":3,
		"current_tx_bytes":4
	}`)

	peersResp := performJSON(t, router, http.MethodGet, "/api/v1/nodes/"+nodeA.NodeID+"/peers", "")
	if peersResp.Code != http.StatusOK {
		t.Fatalf("expected peers HTTP 200, got %d: %s", peersResp.Code, peersResp.Body.String())
	}

	peers := decodeResponseData[protocol.PeersResponse](t, peersResp)
	if len(peers.Peers) != 1 {
		t.Fatalf("expected one peer, got %d", len(peers.Peers))
	}
	peer := peers.Peers[0]
	if peer.TargetVirtualIP != nodeB.VirtualIP || peer.TargetPublicIP != "198.51.100.12" || peer.TargetPublicPort != 45678 {
		t.Fatalf("peer did not match node B heartbeat data: %+v", peer)
	}
	if peer.NATType != "Symmetric" || peer.RecommendMode != protocol.RecommendModeRelay {
		t.Fatalf("expected relay recommendation for symmetric NAT, got %+v", peer)
	}
}

func TestHeartbeatUnknownNodeReturnsAPIError(t *testing.T) {
	router := newTestRouter()

	resp := performJSON(t, router, http.MethodPost, "/api/v1/nodes/not-found/heartbeat", `{
		"nat_type":"Full Cone",
		"public_ip":"203.0.113.5",
		"public_port":54321,
		"current_rx_bytes":0,
		"current_tx_bytes":0
	}`)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d: %s", resp.Code, resp.Body.String())
	}

	envelope := decodeEnvelope(t, resp)
	if envelope.Code != http.StatusNotFound || envelope.Msg != "node not found" || string(envelope.Data) != "null" {
		t.Fatalf("unexpected error envelope: %+v data=%s", envelope, string(envelope.Data))
	}
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(state.NewRegistry())
}

func registerNode(t *testing.T, router http.Handler, machineID, hostname, osName, localIP string) protocol.RegisterNodeResponse {
	t.Helper()

	body := `{
		"machine_id":"` + machineID + `",
		"hostname":"` + hostname + `",
		"os":"` + osName + `",
		"local_ip":"` + localIP + `"
	}`
	resp := performJSON(t, router, http.MethodPost, "/api/v1/nodes/register", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected register HTTP 200, got %d: %s", resp.Code, resp.Body.String())
	}
	return decodeResponseData[protocol.RegisterNodeResponse](t, resp)
}

func performJSON(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeResponseData[T any](t *testing.T, resp *httptest.ResponseRecorder) T {
	t.Helper()

	envelope := decodeEnvelope(t, resp)
	if envelope.Code != protocol.CodeSuccess {
		t.Fatalf("expected API code 200, got %d: %s", envelope.Code, resp.Body.String())
	}
	if envelope.Msg != protocol.MsgSuccess {
		t.Fatalf("expected API msg success, got %q", envelope.Msg)
	}

	var data T
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode response data: %v; body=%s", err, resp.Body.String())
	}
	return data
}

func decodeEnvelope(t *testing.T, resp *httptest.ResponseRecorder) responseEnvelope {
	t.Helper()

	var envelope responseEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response envelope: %v; body=%s", err, resp.Body.String())
	}
	return envelope
}
