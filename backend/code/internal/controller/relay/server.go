package relay

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	netWeaverVersion = 0x01
	netWeaverHeader  = 19

	messageTypeKeepalive    = 0x02
	messageTypeRelayConfirm = 0x20

	sessionIdleTimeout = 2 * time.Minute
)

type Server struct {
	conn *net.UDPConn

	mu       sync.Mutex
	sessions map[uint32]*session
}

type session struct {
	endpoints map[string]endpoint
}

type endpoint struct {
	addr     *net.UDPAddr
	lastSeen time.Time
}

type packet struct {
	messageType  byte
	sessionID    uint32
	srcVirtualIP string
	dstVirtualIP string
}

func Listen(addr string) (*Server, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &Server{
		conn:     conn,
		sessions: make(map[uint32]*session),
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		<-ctx.Done()
		_ = s.conn.Close()
	}()

	buf := make([]byte, 64*1024)
	for {
		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		data := make([]byte, n)
		copy(data, buf[:n])
		if err := s.handlePacket(data, remoteAddr); err != nil {
			log.Printf("udp relay dropped packet from %s: %v", remoteAddr.String(), err)
		}
	}
}

func (s *Server) Close() error {
	return s.conn.Close()
}

func (s *Server) Addr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *Server) handlePacket(data []byte, remoteAddr *net.UDPAddr) error {
	packet, err := decodePacket(data)
	if err != nil {
		return err
	}
	if packet.sessionID == 0 {
		return nil
	}
	if packet.messageType == messageTypeKeepalive {
		if err := s.sendRelayConfirm(packet, remoteAddr); err != nil {
			return err
		}
	}

	targetAddr := s.rememberEndpoint(packet, remoteAddr)
	if targetAddr == nil {
		return nil
	}

	if _, err := s.conn.WriteToUDP(data, targetAddr); err != nil {
		return fmt.Errorf("forward to %s: %w", targetAddr.String(), err)
	}
	return nil
}

func (s *Server) sendRelayConfirm(packet packet, remoteAddr *net.UDPAddr) error {
	data, err := encodePacket(messageTypeRelayConfirm, packet.sessionID, packet.dstVirtualIP, packet.srcVirtualIP, nil)
	if err != nil {
		return err
	}
	if _, err := s.conn.WriteToUDP(data, remoteAddr); err != nil {
		return fmt.Errorf("send relay confirm to %s: %w", remoteAddr.String(), err)
	}
	return nil
}

func (s *Server) rememberEndpoint(packet packet, remoteAddr *net.UDPAddr) *net.UDPAddr {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	sessionState := s.sessions[packet.sessionID]
	if sessionState == nil {
		sessionState = &session{endpoints: make(map[string]endpoint)}
		s.sessions[packet.sessionID] = sessionState
	}

	sessionState.endpoints[packet.srcVirtualIP] = endpoint{
		addr:     cloneUDPAddr(remoteAddr),
		lastSeen: now,
	}
	pruneEndpoints(sessionState, now)

	target, ok := sessionState.endpoints[packet.dstVirtualIP]
	if !ok && packet.dstVirtualIP == "0.0.0.0" {
		target, ok = firstOtherEndpoint(sessionState, packet.srcVirtualIP)
	}
	if !ok || sameUDPAddr(target.addr, remoteAddr) {
		return nil
	}

	return cloneUDPAddr(target.addr)
}

func decodePacket(data []byte) (packet, error) {
	if len(data) < netWeaverHeader {
		return packet{}, fmt.Errorf("packet too short")
	}
	if data[0] != netWeaverVersion {
		return packet{}, fmt.Errorf("unsupported version: %d", data[0])
	}

	payloadLen := int(binary.BigEndian.Uint16(data[15:17]))
	if len(data) < netWeaverHeader+payloadLen {
		return packet{}, fmt.Errorf("truncated payload")
	}

	expectedChecksum := binary.BigEndian.Uint16(data[17:19])
	header := make([]byte, netWeaverHeader)
	copy(header, data[:netWeaverHeader])
	header[17], header[18] = 0, 0
	if actualChecksum := crc16CCITTFalse(header); actualChecksum != expectedChecksum {
		return packet{}, fmt.Errorf("invalid checksum")
	}

	return packet{
		messageType:  data[1],
		sessionID:    binary.BigEndian.Uint32(data[3:7]),
		srcVirtualIP: net.IPv4(data[7], data[8], data[9], data[10]).String(),
		dstVirtualIP: net.IPv4(data[11], data[12], data[13], data[14]).String(),
	}, nil
}

func encodePacket(messageType byte, sessionID uint32, srcVirtualIP string, dstVirtualIP string, payload []byte) ([]byte, error) {
	if len(payload) > 0xffff {
		return nil, fmt.Errorf("payload too large: %d", len(payload))
	}

	srcIP := parseIPv4(srcVirtualIP)
	if srcIP == nil {
		return nil, fmt.Errorf("invalid source virtual ip: %s", srcVirtualIP)
	}
	dstIP := parseIPv4(dstVirtualIP)
	if dstIP == nil {
		return nil, fmt.Errorf("invalid target virtual ip: %s", dstVirtualIP)
	}

	data := make([]byte, netWeaverHeader+len(payload))
	data[0] = netWeaverVersion
	data[1] = messageType
	binary.BigEndian.PutUint32(data[3:7], sessionID)
	copy(data[7:11], srcIP)
	copy(data[11:15], dstIP)
	binary.BigEndian.PutUint16(data[15:17], uint16(len(payload)))
	copy(data[netWeaverHeader:], payload)
	binary.BigEndian.PutUint16(data[17:19], crc16CCITTFalse(data[:netWeaverHeader]))
	return data, nil
}

func parseIPv4(value string) []byte {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.To4()
}

func pruneEndpoints(sessionState *session, now time.Time) {
	for virtualIP, endpoint := range sessionState.endpoints {
		if now.Sub(endpoint.lastSeen) > sessionIdleTimeout {
			delete(sessionState.endpoints, virtualIP)
		}
	}
}

func firstOtherEndpoint(sessionState *session, srcVirtualIP string) (endpoint, bool) {
	for virtualIP, endpoint := range sessionState.endpoints {
		if virtualIP != srcVirtualIP {
			return endpoint, true
		}
	}
	return endpoint{}, false
}

func cloneUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}

	ip := make(net.IP, len(addr.IP))
	copy(ip, addr.IP)
	return &net.UDPAddr{
		IP:   ip,
		Port: addr.Port,
		Zone: addr.Zone,
	}
}

func sameUDPAddr(a *net.UDPAddr, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Port == b.Port && a.Zone == b.Zone && a.IP.Equal(b.IP)
}

func crc16CCITTFalse(data []byte) uint16 {
	var crc uint16 = 0xffff
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
