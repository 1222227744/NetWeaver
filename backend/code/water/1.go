package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/songgao/water"
)

func main() {
	var ifName string
	var bufSize int
	var bootstrap bool
	var autoICMPEchoReply bool
	var ownerID int
	var groupID int

	flag.StringVar(&ifName, "ifname", "tuno", "TUN interface name")
	flag.IntVar(&bufSize, "buf", 2000, "read buffer size")
	flag.BoolVar(&bootstrap, "bootstrap", false, "create a persistent TUN and set owner/group for non-root runs")
	flag.BoolVar(&autoICMPEchoReply, "icmp-echo-reply", true, "auto-reply ICMP echo requests received on TUN")
	flag.IntVar(&ownerID, "owner", -1, "owner uid used with -bootstrap (default current user or SUDO_UID)")
	flag.IntVar(&groupID, "group", -1, "group gid used with -bootstrap (default current group or SUDO_GID)")
	flag.Parse()

	if bufSize <= 0 {
		log.Fatalf("invalid -buf: %d", bufSize)
	}

	params := water.PlatformSpecificParams{
		Name: ifName,
	}
	if bootstrap {
		targetUID, targetGID := defaultTargetOwnerGroup()
		if ownerID >= 0 {
			targetUID = ownerID
		}
		if groupID >= 0 {
			targetGID = groupID
		}

		params.Persist = true
		params.Permissions = &water.DevicePermissions{
			Owner: uint(targetUID),
			Group: uint(targetGID),
		}

		log.Printf("bootstrap mode: persist=%t owner=%d group=%d ifname=%s", params.Persist, targetUID, targetGID, ifName)
	}

	cfg := water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: params,
	}

	var dev io.ReadWriteCloser
	var devName string

	ifce, err := water.New(cfg)
	if err != nil {
		// water Linux implementation always touches TUNSETPERSIST. For a persistent
		// device opened by non-root user, this may fail with EPERM even though
		// TUNSETIFF itself could succeed. Fallback to direct-open path in that case.
		if !bootstrap && isPermissionErr(err) {
			f, name, directErr := openTunDirect(ifName)
			if directErr == nil {
				log.Printf("water.New hit permission error (%v), fallback direct-open succeeded", err)
				dev = f
				devName = name
			} else {
				log.Fatalf("create/open TUN failed: water=%v, direct=%v\n%s", err, directErr, tunCreateHints(err, ifName))
			}
		} else {
			log.Fatalf("create/open TUN failed: %v\n%s", err, tunCreateHints(err, ifName))
		}
	} else {
		dev = ifce
		devName = ifce.Name()
	}
	if bootstrap {
		defer dev.Close()
		log.Printf("bootstrap success: %s is persistent now", devName)
		log.Printf("one-time network setup (root):")
		log.Printf("  sudo ip addr add 10.23.0.1/24 dev %s", devName)
		log.Printf("  sudo ip link set %s up", devName)
		log.Printf("then run as normal user:")
		log.Printf("  /usr/local/go/bin/go run . -ifname %s", devName)
		return
	}
	defer dev.Close()

	log.Printf("TUN ready: %s", devName)
	log.Printf("Linux setup example:")
	log.Printf("  sudo ip addr add 10.23.0.1/24 dev %s", devName)
	log.Printf("  sudo ip link set %s up", devName)
	log.Printf("  ping -I %s -c 4 10.23.0.2", devName)

	filterNet, err := findInterfaceIPv4Subnet(devName)
	if err != nil {
		log.Printf("subnet filter disabled: %v", err)
		log.Printf("capture mode: dump all IP packets seen on %s", devName)
	} else {
		log.Printf("capture mode: only dump packets whose IPv4 dst is in %s", filterNet.String())
	}
	if autoICMPEchoReply {
		log.Printf("ICMP echo auto-reply: enabled")
	} else {
		log.Printf("ICMP echo auto-reply: disabled")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println()
		log.Println("signal received, exiting")
		_ = dev.Close()
		os.Exit(0)
	}()

	buf := make([]byte, bufSize)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			log.Fatalf("read failed: %v", err)
		}

		packet := buf[:n]
		if autoICMPEchoReply {
			replied, err := replyICMPEchoIfNeeded(dev, packet)
			if err != nil {
				log.Printf("send ICMP echo reply failed: %v", err)
			} else if replied {
				log.Printf("ICMP echo reply sent")
			}
		}

		if filterNet != nil {
			dst, ok := packetIPv4Dst(packet)
			if !ok || !filterNet.Contains(dst) {
				continue
			}
		}

		fmt.Printf("\n[%s] packet %d bytes\n", time.Now().Format("15:04:05.000"), n)
		fmt.Print(hex.Dump(packet))
	}
}

func isPermissionErr(err error) bool {
	return errors.Is(err, syscall.EPERM) || errors.Is(err, os.ErrPermission)
}

const (
	cIFFTUN  = 0x0001
	cIFFNOPI = 0x1000
)

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	Pad   [0x28 - 0x10 - 2]byte
}

func openTunDirect(ifName string) (io.ReadWriteCloser, string, error) {
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, "", err
	}

	var req ifReq
	req.Flags = cIFFTUN | cIFFNOPI
	copy(req.Name[:], ifName)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		_ = syscall.Close(fd)
		return nil, "", os.NewSyscallError("ioctl", errno)
	}

	name := strings.Trim(string(req.Name[:]), "\x00")
	return os.NewFile(uintptr(fd), "tun"), name, nil
}

func defaultTargetOwnerGroup() (uid int, gid int) {
	uid = os.Getuid()
	gid = os.Getgid()

	if os.Geteuid() == 0 {
		if v, ok := parsePositiveIntEnv("SUDO_UID"); ok {
			uid = v
		}
		if v, ok := parsePositiveIntEnv("SUDO_GID"); ok {
			gid = v
		}
	}
	return uid, gid
}

func parsePositiveIntEnv(key string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

func tunCreateHints(err error, ifName string) string {
	var b strings.Builder
	b.WriteString("How to fix:\n")

	switch {
	case errors.Is(err, syscall.EPERM), errors.Is(err, os.ErrPermission):
		b.WriteString("1) Recommended one-time bootstrap (root), then run without sudo:\n")
		b.WriteString(fmt.Sprintf("   `sudo /usr/local/go/bin/go run . -bootstrap -ifname %s`\n", ifName))
		b.WriteString(fmt.Sprintf("   `/usr/local/go/bin/go run . -ifname %s`\n", ifName))
		b.WriteString("2) Or run with NET_ADMIN privilege each time:\n")
		b.WriteString("   `sudo /usr/local/go/bin/go run .`\n")
		b.WriteString("3) Or build once and grant capability to the binary (go run temp binaries cannot keep this capability):\n")
		b.WriteString("   `go build -o tun-sniffer .`\n")
		b.WriteString("   `sudo setcap cap_net_admin+ep ./tun-sniffer`\n")
		b.WriteString("   `./tun-sniffer`\n")
		b.WriteString("4) In Docker, start container with: `--cap-add=NET_ADMIN --device /dev/net/tun`\n")
	case errors.Is(err, syscall.ENOENT):
		b.WriteString("1) `/dev/net/tun` is missing. Ensure TUN module is loaded:\n")
		b.WriteString("   `sudo modprobe tun`\n")
		b.WriteString("2) If in container, also mount tun device: `--device /dev/net/tun`\n")
	default:
		b.WriteString("Check kernel TUN support, container security policy, and NET_ADMIN privilege.\n")
	}

	envHints := runtimeCapabilityHints()
	if envHints != "" {
		b.WriteString("\nEnvironment diagnostics:\n")
		b.WriteString(envHints)
	}

	b.WriteString(fmt.Sprintf("If using an existing persistent tun, verify `%s` ownership/permissions are assigned to your user.\n", ifName))
	return b.String()
}

func runtimeCapabilityHints() string {
	const capNetAdminBit = 12
	const capNetAdminMask uint64 = 1 << capNetAdminBit

	capEff, capEffOK := readProcStatusHex("CapEff")
	capBnd, capBndOK := readProcStatusHex("CapBnd")
	inWSL := isLikelyWSL()

	var b strings.Builder
	if capEffOK {
		b.WriteString(fmt.Sprintf("- CapEff=0x%x\n", capEff))
	}
	if capBndOK {
		b.WriteString(fmt.Sprintf("- CapBnd=0x%x\n", capBnd))
	}
	if inWSL {
		b.WriteString("- WSL kernel detected\n")
	}

	if capBndOK && (capBnd&capNetAdminMask) == 0 {
		b.WriteString("- CAP_NET_ADMIN is not in bounding set; this runtime cannot grant NET_ADMIN to your process.\n")
		b.WriteString("  Run this program in a host/container/VM that exposes CAP_NET_ADMIN.\n")
	} else if capEffOK && (capEff&capNetAdminMask) == 0 {
		b.WriteString("- CAP_NET_ADMIN not effective for current process. Try sudo or setcap on a built binary.\n")
	}

	return b.String()
}

func readProcStatusHex(key string) (uint64, bool) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	prefix := key + ":"
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		v, err := strconv.ParseUint(raw, 16, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

func isLikelyWSL() bool {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	l := strings.ToLower(string(data))
	return strings.Contains(l, "microsoft") || strings.Contains(l, "wsl")
}

func findInterfaceIPv4Subnet(ifName string) (*net.IPNet, error) {
	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("find interface %q failed: %w", ifName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("read interface addresses failed: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ipNet.IP.To4()
		if v4 == nil {
			continue
		}
		return &net.IPNet{IP: v4.Mask(ipNet.Mask), Mask: ipNet.Mask}, nil
	}

	return nil, fmt.Errorf("no IPv4 address found on %s", ifName)
}

func packetIPv4Dst(packet []byte) (net.IP, bool) {
	if len(packet) < 20 {
		return nil, false
	}
	if packet[0]>>4 != 4 {
		return nil, false
	}
	ihl := int(packet[0]&0x0f) * 4
	if ihl < 20 || len(packet) < ihl {
		return nil, false
	}
	return net.IPv4(packet[16], packet[17], packet[18], packet[19]).To4(), true
}

func replyICMPEchoIfNeeded(dev io.Writer, packet []byte) (bool, error) {
	if len(packet) < 20 {
		return false, nil
	}
	if version := packet[0] >> 4; version != 4 {
		return false, nil
	}

	ihl := int(packet[0]&0x0f) * 4
	if ihl < 20 || len(packet) < ihl+8 {
		return false, nil
	}
	if packet[9] != 1 { // IPv4 ICMP
		return false, nil
	}

	totalLen := int(binary.BigEndian.Uint16(packet[2:4]))
	if totalLen <= 0 || totalLen > len(packet) {
		totalLen = len(packet)
	}
	if totalLen < ihl+8 {
		return false, nil
	}

	fragOffset := binary.BigEndian.Uint16(packet[6:8]) & 0x1fff
	if fragOffset != 0 {
		return false, nil
	}

	icmp := packet[ihl:totalLen]
	if icmp[0] != 8 || icmp[1] != 0 { // echo request
		return false, nil
	}

	reply := make([]byte, totalLen)
	copy(reply, packet[:totalLen])

	// Swap IPv4 source and destination.
	copy(reply[12:16], packet[16:20])
	copy(reply[16:20], packet[12:16])
	reply[8] = 64 // fresh TTL for generated reply

	// Rewrite ICMP type + checksums.
	reply[ihl] = 0 // echo reply
	reply[ihl+1] = 0
	reply[ihl+2] = 0
	reply[ihl+3] = 0
	icmpCsum := checksum16(reply[ihl:totalLen])
	binary.BigEndian.PutUint16(reply[ihl+2:ihl+4], icmpCsum)

	reply[10] = 0
	reply[11] = 0
	ipCsum := checksum16(reply[:ihl])
	binary.BigEndian.PutUint16(reply[10:12], ipCsum)

	_, err := dev.Write(reply)
	if err != nil {
		return false, err
	}
	return true, nil
}

func checksum16(b []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(b); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(b[i : i+2]))
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}
	for (sum >> 16) != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
