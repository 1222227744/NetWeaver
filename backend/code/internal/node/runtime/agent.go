package runtime

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"netweaver-backend/internal/node/client"
	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

type Config struct {
	ControllerURL     string
	MachineID         string
	Hostname          string
	LocalIP           string
	NATType           string
	PublicIP          string
	PublicPort        int
	HeartbeatInterval time.Duration
	PeerInterval      time.Duration
}

type Agent struct {
	cfg       Config
	client    *client.Client
	nodeID    string
	virtualIP string
}

func Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("node run", flag.ContinueOnError)
	controllerURL := fs.String("controller", config.DefaultControllerURL, "controller base URL")
	machineID := fs.String("machine-id", "", "stable machine id; default reads /etc/machine-id or derives one")
	hostname := fs.String("hostname", "", "node hostname; default uses os.Hostname")
	localIP := fs.String("local-ip", "", "node local IP; default detects the outbound interface IP")
	natType := fs.String("nat-type", protocol.DefaultNATType, "NAT type reported to controller")
	publicIP := fs.String("public-ip", "", "public IP reported in heartbeat; default lets controller infer it")
	publicPort := fs.Int("public-port", 0, "public port reported in heartbeat")
	heartbeatInterval := fs.Duration("interval", config.DefaultHeartbeatInterval, "heartbeat interval")
	peerInterval := fs.Duration("peer-interval", config.DefaultHeartbeatInterval, "peer sync interval")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *heartbeatInterval <= 0 {
		return fmt.Errorf("invalid -interval: %s", heartbeatInterval.String())
	}
	if *peerInterval <= 0 {
		return fmt.Errorf("invalid -peer-interval: %s", peerInterval.String())
	}

	resolvedHostname := strings.TrimSpace(*hostname)
	if resolvedHostname == "" {
		resolvedHostname = defaultHostname()
	}

	resolvedLocalIP := strings.TrimSpace(*localIP)
	if resolvedLocalIP == "" {
		resolvedLocalIP = defaultLocalIP()
	}

	resolvedMachineID := strings.TrimSpace(*machineID)
	if resolvedMachineID == "" {
		resolvedMachineID = defaultMachineID(resolvedHostname, resolvedLocalIP)
	}

	agent := New(Config{
		ControllerURL:     strings.TrimSpace(*controllerURL),
		MachineID:         resolvedMachineID,
		Hostname:          resolvedHostname,
		LocalIP:           resolvedLocalIP,
		NATType:           strings.TrimSpace(*natType),
		PublicIP:          strings.TrimSpace(*publicIP),
		PublicPort:        *publicPort,
		HeartbeatInterval: *heartbeatInterval,
		PeerInterval:      *peerInterval,
	})

	return agent.Run(ctx)
}

func New(cfg Config) *Agent {
	if cfg.ControllerURL == "" {
		cfg.ControllerURL = config.DefaultControllerURL
	}
	if cfg.NATType == "" {
		cfg.NATType = protocol.DefaultNATType
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = config.DefaultHeartbeatInterval
	}
	if cfg.PeerInterval <= 0 {
		cfg.PeerInterval = cfg.HeartbeatInterval
	}

	return &Agent{
		cfg:    cfg,
		client: client.New(cfg.ControllerURL),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	registerResp, err := a.register(ctx)
	if err != nil {
		return err
	}

	a.nodeID = registerResp.NodeID
	a.virtualIP = registerResp.VirtualIP
	if registerResp.KeepaliveInterval > 0 && a.cfg.HeartbeatInterval == config.DefaultHeartbeatInterval {
		a.cfg.HeartbeatInterval = time.Duration(registerResp.KeepaliveInterval) * time.Second
	}

	log.Printf("node registered: node_id=%s virtual_ip=%s controller=%s", a.nodeID, a.virtualIP, a.cfg.ControllerURL)

	if err := a.sendHeartbeat(ctx); err != nil {
		log.Printf("initial heartbeat failed: %v", err)
	}
	a.syncPeers(ctx)

	heartbeatTicker := time.NewTicker(a.cfg.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	peerTicker := time.NewTicker(a.cfg.PeerInterval)
	defer peerTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			if err := a.sendHeartbeat(ctx); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-peerTicker.C:
			a.syncPeers(ctx)
		}
	}
}

func (a *Agent) register(ctx context.Context) (protocol.RegisterNodeResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, config.DefaultRequestTimeout)
	defer cancel()

	return a.client.RegisterNode(reqCtx, protocol.RegisterNodeRequest{
		MachineID: a.cfg.MachineID,
		Hostname:  a.cfg.Hostname,
		OS:        goruntime.GOOS,
		LocalIP:   a.cfg.LocalIP,
	})
}

func (a *Agent) sendHeartbeat(ctx context.Context) error {
	rxBytes, txBytes := readSystemTrafficBytes()

	reqCtx, cancel := context.WithTimeout(ctx, config.DefaultRequestTimeout)
	defer cancel()

	resp, err := a.client.Heartbeat(reqCtx, a.nodeID, protocol.HeartbeatRequest{
		NATType:        a.cfg.NATType,
		PublicIP:       a.cfg.PublicIP,
		PublicPort:     a.cfg.PublicPort,
		CurrentRXBytes: rxBytes,
		CurrentTXBytes: txBytes,
	})
	if err != nil {
		return err
	}

	log.Printf("heartbeat ok: node_id=%s peers=%d action=%s rx=%d tx=%d", a.nodeID, resp.ConnectedPeers, resp.ActionRequired, rxBytes, txBytes)
	if resp.ActionRequired == protocol.ActionSyncPeers {
		a.syncPeers(ctx)
	}

	return nil
}

func (a *Agent) syncPeers(ctx context.Context) {
	if a.nodeID == "" {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, config.DefaultRequestTimeout)
	defer cancel()

	peers, err := a.client.GetPeers(reqCtx, a.nodeID)
	if err != nil {
		log.Printf("sync peers failed: %v", err)
		return
	}

	if len(peers.Peers) == 0 {
		log.Printf("peers synced: none")
		return
	}

	summary := make([]string, 0, len(peers.Peers))
	for _, peer := range peers.Peers {
		summary = append(summary, fmt.Sprintf("%s(%s,%s)", peer.TargetHostname, peer.TargetVirtualIP, peer.RecommendMode))
	}
	log.Printf("peers synced: %s", strings.Join(summary, ", "))
}

func defaultHostname() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "netweaver-node"
	}
	return strings.TrimSpace(hostname)
}

func defaultLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && localAddr.IP != nil {
			return localAddr.IP.String()
		}
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip == nil {
				continue
			}
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
	}

	return "127.0.0.1"
}

func defaultMachineID(hostname string, localIP string) string {
	if machineID := readFirstLine("/etc/machine-id"); machineID != "" {
		return machineID
	}

	seedParts := []string{hostname, localIP}
	if hardwareAddr := firstHardwareAddr(); hardwareAddr != "" {
		seedParts = append(seedParts, hardwareAddr)
	}

	sum := sha1.Sum([]byte(strings.Join(seedParts, "|")))
	return "derived-" + hex.EncodeToString(sum[:])[:16]
}

func readFirstLine(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return ""
	}

	return strings.TrimSpace(scanner.Text())
}

func firstHardwareAddr() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}

	return ""
}

func readSystemTrafficBytes() (uint64, uint64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var rxTotal uint64
	var txTotal uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		ifaceName := strings.TrimSpace(parts[0])
		if ifaceName == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		rxBytes, rxErr := strconv.ParseUint(fields[0], 10, 64)
		txBytes, txErr := strconv.ParseUint(fields[8], 10, 64)
		if rxErr != nil || txErr != nil {
			continue
		}

		rxTotal += rxBytes
		txTotal += txBytes
	}

	return rxTotal, txTotal
}
