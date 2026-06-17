package runtime

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	"netweaver-backend/pkg/protocol"
)

const (
	stunBindingRequest  = 0x0001
	stunBindingSuccess  = 0x0101
	stunMagicCookie     = 0x2112a442
	stunAttrMappedAddr  = 0x0001
	stunAttrXORMapped   = 0x0020
	stunRequestTimeout  = 3 * time.Second
	stunRequestAttempts = 3
)

type NATResult struct {
	NATType    string
	PublicIP   string
	PublicPort int
}

type mappedAddress struct {
	IP   string
	Port int
}

func DetectNAT(ctx context.Context, conn *net.UDPConn, servers []string, localIP string) NATResult {
	result := NATResult{NATType: protocol.DefaultNATType}
	if conn == nil || len(servers) == 0 {
		return result
	}
	defer func() {
		_ = conn.SetReadDeadline(time.Time{})
	}()

	mapped := make([]mappedAddress, 4)
	success := make([]bool, 4)

	for i := 0; i < len(servers) && i < 3; i++ {
		addr, err := net.ResolveUDPAddr("udp", servers[i])
		if err != nil {
			log.Printf("resolve stun server failed: server=%s err=%v", servers[i], err)
			continue
		}
		if mappedAddr, ok := stunBinding(ctx, conn, addr); ok {
			mapped[i] = mappedAddr
			success[i] = true
		}
	}

	if len(servers) > 0 {
		if addr, err := net.ResolveUDPAddr("udp", servers[0]); err == nil {
			if secondaryConn, listenErr := net.ListenUDP("udp", nil); listenErr == nil {
				if mappedAddr, ok := stunBinding(ctx, secondaryConn, addr); ok {
					mapped[3] = mappedAddr
					success[3] = true
				}
				_ = secondaryConn.Close()
			}
		}
	}

	result.NATType = classifyNAT(conn, localIP, mapped, success)
	if first, ok := firstMappedAddress(mapped, success); ok {
		result.PublicIP = first.IP
		result.PublicPort = first.Port
	}
	return result
}

func stunBinding(ctx context.Context, conn *net.UDPConn, addr *net.UDPAddr) (mappedAddress, bool) {
	for attempt := 0; attempt < stunRequestAttempts; attempt++ {
		transactionID, request, err := newSTUNBindingRequest()
		if err != nil {
			return mappedAddress{}, false
		}

		if _, err := conn.WriteToUDP(request, addr); err != nil {
			continue
		}

		deadline := time.Now().Add(stunRequestTimeout)
		if err := conn.SetReadDeadline(deadline); err != nil {
			return mappedAddress{}, false
		}

		buf := make([]byte, 1500)
		for {
			select {
			case <-ctx.Done():
				return mappedAddress{}, false
			default:
			}

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				break
			}

			mappedAddr, ok := parseSTUNBindingResponse(buf[:n], transactionID)
			if ok {
				return mappedAddr, true
			}
			if time.Now().After(deadline) {
				break
			}
		}
	}

	return mappedAddress{}, false
}

func newSTUNBindingRequest() ([]byte, []byte, error) {
	transactionID := make([]byte, 12)
	if _, err := rand.Read(transactionID); err != nil {
		return nil, nil, err
	}

	request := make([]byte, 20)
	binary.BigEndian.PutUint16(request[0:2], stunBindingRequest)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint32(request[4:8], stunMagicCookie)
	copy(request[8:20], transactionID)
	return transactionID, request, nil
}

func parseSTUNBindingResponse(data []byte, transactionID []byte) (mappedAddress, bool) {
	if len(data) < 20 {
		return mappedAddress{}, false
	}
	if binary.BigEndian.Uint16(data[0:2]) != stunBindingSuccess {
		return mappedAddress{}, false
	}
	if binary.BigEndian.Uint32(data[4:8]) != stunMagicCookie {
		return mappedAddress{}, false
	}
	if !bytes.Equal(data[8:20], transactionID) {
		return mappedAddress{}, false
	}

	msgLen := int(binary.BigEndian.Uint16(data[2:4]))
	if msgLen < 0 || 20+msgLen > len(data) {
		return mappedAddress{}, false
	}

	for offset := 20; offset+4 <= 20+msgLen; {
		attrType := binary.BigEndian.Uint16(data[offset : offset+2])
		attrLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		valueStart := offset + 4
		valueEnd := valueStart + attrLen
		if valueEnd > len(data) {
			return mappedAddress{}, false
		}

		switch attrType {
		case stunAttrXORMapped:
			if mappedAddr, ok := parseXORMappedAddress(data[valueStart:valueEnd]); ok {
				return mappedAddr, true
			}
		case stunAttrMappedAddr:
			if mappedAddr, ok := parseMappedAddress(data[valueStart:valueEnd]); ok {
				return mappedAddr, true
			}
		}

		offset = valueEnd
		if padding := offset % 4; padding != 0 {
			offset += 4 - padding
		}
	}

	return mappedAddress{}, false
}

func parseXORMappedAddress(value []byte) (mappedAddress, bool) {
	if len(value) < 8 || value[1] != 0x01 {
		return mappedAddress{}, false
	}

	port := int(binary.BigEndian.Uint16(value[2:4]) ^ uint16(stunMagicCookie>>16))
	cookie := make([]byte, 4)
	binary.BigEndian.PutUint32(cookie, stunMagicCookie)
	ip := net.IPv4(value[4]^cookie[0], value[5]^cookie[1], value[6]^cookie[2], value[7]^cookie[3]).String()
	return mappedAddress{IP: ip, Port: port}, true
}

func parseMappedAddress(value []byte) (mappedAddress, bool) {
	if len(value) < 8 || value[1] != 0x01 {
		return mappedAddress{}, false
	}

	port := int(binary.BigEndian.Uint16(value[2:4]))
	ip := net.IPv4(value[4], value[5], value[6], value[7]).String()
	return mappedAddress{IP: ip, Port: port}, true
}

func classifyNAT(conn *net.UDPConn, localIP string, mapped []mappedAddress, success []bool) string {
	if !success[0] && !success[1] && !success[2] {
		return protocol.DefaultNATType
	}

	if success[0] && matchesLocalEndpoint(conn, localIP, mapped[0]) {
		return "Full Cone"
	}
	if success[0] && success[1] && success[2] && sameMappedAddress(mapped[0], mapped[1]) && sameMappedAddress(mapped[1], mapped[2]) {
		return "Restricted Cone"
	}
	if success[0] && success[1] && success[2] && !sameMappedAddress(mapped[0], mapped[2]) {
		return "Symmetric"
	}
	if success[0] && success[1] && !success[2] && success[3] {
		return "Port Restricted Cone"
	}
	return protocol.DefaultNATType
}

func firstMappedAddress(mapped []mappedAddress, success []bool) (mappedAddress, bool) {
	for i, ok := range success {
		if ok {
			return mapped[i], true
		}
	}
	return mappedAddress{}, false
}

func sameMappedAddress(a mappedAddress, b mappedAddress) bool {
	return a.IP == b.IP && a.Port == b.Port
}

func matchesLocalEndpoint(conn *net.UDPConn, localIP string, mapped mappedAddress) bool {
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || localAddr == nil {
		return false
	}

	ip := net.ParseIP(localIP).To4()
	if ip == nil {
		return false
	}
	return mapped.IP == ip.String() && mapped.Port == localAddr.Port
}

func (m mappedAddress) String() string {
	return fmt.Sprintf("%s:%d", m.IP, m.Port)
}
