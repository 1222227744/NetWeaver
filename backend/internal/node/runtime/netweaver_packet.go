package runtime

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	netWeaverVersion = 0x01
	netWeaverHeader  = 19

	messageTypeData         = 0x01
	messageTypeKeepalive    = 0x02
	messageTypeKeepaliveAck = 0x03
	messageTypePunchProbe   = 0x10
	messageTypePunchAck     = 0x11
	messageTypeRelayConfirm = 0x20
)

type netWeaverPacket struct {
	messageType  byte
	sessionID    uint32
	srcVirtualIP string
	dstVirtualIP string
	payload      []byte
}

func encodeNetWeaverPacket(messageType byte, sessionID uint32, srcVirtualIP string, dstVirtualIP string, payload []byte) ([]byte, error) {
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

	packet := make([]byte, netWeaverHeader+len(payload))
	packet[0] = netWeaverVersion
	packet[1] = messageType
	packet[2] = 0
	binary.BigEndian.PutUint32(packet[3:7], sessionID)
	copy(packet[7:11], srcIP)
	copy(packet[11:15], dstIP)
	binary.BigEndian.PutUint16(packet[15:17], uint16(len(payload)))
	copy(packet[netWeaverHeader:], payload)

	checksum := crc16CCITTFalse(packet[:netWeaverHeader])
	binary.BigEndian.PutUint16(packet[17:19], checksum)
	return packet, nil
}

func decodeNetWeaverPacket(data []byte) (netWeaverPacket, error) {
	if len(data) < netWeaverHeader {
		return netWeaverPacket{}, fmt.Errorf("packet too short")
	}
	if data[0] != netWeaverVersion {
		return netWeaverPacket{}, fmt.Errorf("unsupported version: %d", data[0])
	}

	payloadLen := int(binary.BigEndian.Uint16(data[15:17]))
	if len(data) < netWeaverHeader+payloadLen {
		return netWeaverPacket{}, fmt.Errorf("truncated payload")
	}

	expectedChecksum := binary.BigEndian.Uint16(data[17:19])
	header := make([]byte, netWeaverHeader)
	copy(header, data[:netWeaverHeader])
	header[17], header[18] = 0, 0
	if actualChecksum := crc16CCITTFalse(header); actualChecksum != expectedChecksum {
		return netWeaverPacket{}, fmt.Errorf("invalid checksum")
	}

	payload := make([]byte, payloadLen)
	copy(payload, data[netWeaverHeader:netWeaverHeader+payloadLen])

	return netWeaverPacket{
		messageType:  data[1],
		sessionID:    binary.BigEndian.Uint32(data[3:7]),
		srcVirtualIP: net.IPv4(data[7], data[8], data[9], data[10]).String(),
		dstVirtualIP: net.IPv4(data[11], data[12], data[13], data[14]).String(),
		payload:      payload,
	}, nil
}

func parseIPv4(value string) []byte {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.To4()
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
