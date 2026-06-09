package runtime

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"netweaver-backend/internal/node/client"
	nodetun "netweaver-backend/internal/node/tun"
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
	STUNServers       []string
	DataListenAddr    string
	PSK               string
	TUNEnabled        bool
	TUNName           string
	TUNBufferSize     int
	HeartbeatInterval time.Duration
	PeerInterval      time.Duration
}

type Agent struct {
	cfg       Config
	client    *client.Client
	nodeID    string
	virtualIP string
	dataConn  *net.UDPConn
	linker    *linkManager
	tunMu     sync.RWMutex
	tunDev    io.ReadWriteCloser
	tunName   string
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
	stunServers := fs.String("stun-servers", config.DefaultSTUNServers, "comma-separated STUN endpoints")
	dataListenAddr := fs.String("data-addr", "0.0.0.0:0", "UDP address used for STUN, P2P punch and data plane")
	psk := fs.String("psk", config.NodePSK(), "node pre-shared key for controller node APIs")
	tunEnabled := fs.Bool("tun", true, "enable TUN data plane in run mode")
	tunName := fs.String("tun-name", config.DefaultTUNName, "TUN interface used for data plane")
	tunBufferSize := fs.Int("tun-buf", 65535, "TUN packet read buffer size")
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
	if *tunBufferSize <= 0 {
		return fmt.Errorf("invalid -tun-buf: %d", *tunBufferSize)
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
		STUNServers:       splitCSV(*stunServers),
		DataListenAddr:    strings.TrimSpace(*dataListenAddr),
		PSK:               strings.TrimSpace(*psk),
		TUNEnabled:        *tunEnabled,
		TUNName:           strings.TrimSpace(*tunName),
		TUNBufferSize:     *tunBufferSize,
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
	if cfg.DataListenAddr == "" {
		cfg.DataListenAddr = "0.0.0.0:0"
	}
	if cfg.PSK == "" {
		cfg.PSK = config.NodePSK()
	}
	if cfg.TUNName == "" {
		cfg.TUNName = config.DefaultTUNName
	}
	if cfg.TUNBufferSize <= 0 {
		cfg.TUNBufferSize = 65535
	}

	agent := &Agent{
		cfg:    cfg,
		client: client.NewWithPSK(cfg.ControllerURL, cfg.PSK),
	}
	agent.linker = newLinkManager(agent)
	return agent
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

	if err := a.openDataSocket(ctx); err != nil {
		log.Printf("data socket disabled: %v", err)
	} else {
		a.detectNAT(ctx)
		go a.linker.readLoop(ctx, a.dataConn)
		go a.linker.probeLoop(ctx)
		a.startTUNDataPlane(ctx)
	}

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

func (a *Agent) startTUNDataPlane(ctx context.Context) {
	if !a.cfg.TUNEnabled {
		log.Printf("TUN data plane disabled")
		return
	}
	if a.dataConn == nil {
		log.Printf("TUN data plane disabled: data socket is not open")
		return
	}

	device, err := nodetun.Open(nodetun.OpenOptions{
		Name: a.cfg.TUNName,
	})
	if err != nil {
		log.Printf("TUN data plane disabled: %v", err)
		return
	}

	a.tunMu.Lock()
	a.tunDev = device.ReadWriteCloser
	a.tunName = device.Name
	a.tunMu.Unlock()

	log.Printf("TUN data plane enabled: ifname=%s virtual_ip=%s", device.Name, a.virtualIP)
	log.Printf("TUN setup reminder: sudo ip addr add %s/16 dev %s && sudo ip link set %s up", a.virtualIP, device.Name, device.Name)

	go func() {
		<-ctx.Done()
		_ = device.Close()
	}()
	go a.tunReadLoop(ctx, device)
}

func (a *Agent) tunReadLoop(ctx context.Context, device *nodetun.Device) {
	buf := make([]byte, a.cfg.TUNBufferSize)
	for {
		n, err := device.Read(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("TUN read failed: %v", err)
				return
			}
		}
		if n == 0 {
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])
		if err := a.linker.sendData(packet); err != nil {
			log.Printf("drop outbound TUN packet: %v", err)
		}
	}
}

func (a *Agent) writeTUN(packet []byte) error {
	a.tunMu.RLock()
	dev := a.tunDev
	name := a.tunName
	a.tunMu.RUnlock()

	if dev == nil {
		return fmt.Errorf("TUN data plane is not open")
	}
	if _, err := dev.Write(packet); err != nil {
		return fmt.Errorf("write TUN %s: %w", name, err)
	}
	return nil
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
	connectedPeerIDs, linkMetrics := a.linker.snapshot()

	reqCtx, cancel := context.WithTimeout(ctx, config.DefaultRequestTimeout)
	defer cancel()

	resp, err := a.client.Heartbeat(reqCtx, a.nodeID, protocol.HeartbeatRequest{
		NATType:          a.cfg.NATType,
		PublicIP:         a.cfg.PublicIP,
		PublicPort:       a.cfg.PublicPort,
		CurrentRXBytes:   rxBytes,
		CurrentTXBytes:   txBytes,
		ConnectedPeers:   len(connectedPeerIDs),
		ConnectedPeerIDs: connectedPeerIDs,
		LinkMetrics:      linkMetrics,
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
		a.linker.replacePeers(nil)
		log.Printf("peers synced: none")
		return
	}

	a.linker.replacePeers(peers.Peers)
	a.linker.establishPeers(ctx, peers.Peers)

	summary := make([]string, 0, len(peers.Peers))
	for _, peer := range peers.Peers {
		summary = append(summary, fmt.Sprintf("%s(%s,%s)", peer.TargetHostname, peer.TargetVirtualIP, peer.RecommendMode))
	}
	log.Printf("peers synced: %s", strings.Join(summary, ", "))
}

func (a *Agent) openDataSocket(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", a.cfg.DataListenAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	a.dataConn = conn
	log.Printf("data socket listening: %s", conn.LocalAddr().String())
	return nil
}

func (a *Agent) detectNAT(ctx context.Context) {
	if a.dataConn == nil {
		return
	}
	if a.cfg.PublicIP != "" && a.cfg.PublicPort > 0 {
		log.Printf("using configured public endpoint: nat_type=%s public=%s:%d", a.cfg.NATType, a.cfg.PublicIP, a.cfg.PublicPort)
		return
	}

	result := DetectNAT(ctx, a.dataConn, a.cfg.STUNServers, a.cfg.LocalIP)
	a.cfg.NATType = result.NATType
	a.cfg.PublicIP = result.PublicIP
	a.cfg.PublicPort = result.PublicPort
	log.Printf("stun result: nat_type=%s public=%s:%d", a.cfg.NATType, a.cfg.PublicIP, a.cfg.PublicPort)
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

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
