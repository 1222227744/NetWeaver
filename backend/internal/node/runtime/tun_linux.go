package runtime

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"netweaver-backend/pkg/config"
)

func defaultTUNPrefixLen() int {
	_, network, err := net.ParseCIDR(config.DefaultVirtualSubnetCIDR)
	if err != nil || network == nil {
		return 16
	}

	prefix, _ := network.Mask.Size()
	if prefix <= 0 {
		return 16
	}
	return prefix
}

func configureTUNInterface(ifName string, virtualIP string, prefixLen int) error {
	if strings.TrimSpace(ifName) == "" {
		return fmt.Errorf("tun interface name is required for auto-config")
	}
	if strings.TrimSpace(virtualIP) == "" {
		return fmt.Errorf("virtual IP is required for auto-config")
	}
	if prefixLen <= 0 || prefixLen > 32 {
		return fmt.Errorf("invalid TUN prefix length %d", prefixLen)
	}

	if err := runIPCommand("addr", "replace", fmt.Sprintf("%s/%d", virtualIP, prefixLen), "dev", ifName); err != nil {
		return fmt.Errorf("assign %s/%d to %s: %w", virtualIP, prefixLen, ifName, err)
	}
	if err := runIPCommand("link", "set", "dev", ifName, "up"); err != nil {
		return fmt.Errorf("bring up %s: %w", ifName, err)
	}
	return nil
}

func cleanupTUNInterface(ifName string, virtualIP string, prefixLen int) error {
	if strings.TrimSpace(ifName) == "" || strings.TrimSpace(virtualIP) == "" || prefixLen <= 0 {
		return nil
	}

	addrErr := runIPCommand("addr", "del", fmt.Sprintf("%s/%d", virtualIP, prefixLen), "dev", ifName)
	linkErr := runIPCommand("link", "set", "dev", ifName, "down")

	if addrErr != nil && !isIgnorableIPCommandError(addrErr.Error()) {
		return fmt.Errorf("remove %s/%d from %s: %w", virtualIP, prefixLen, ifName, addrErr)
	}
	if linkErr != nil && !isIgnorableIPCommandError(linkErr.Error()) {
		return fmt.Errorf("bring down %s: %w", ifName, linkErr)
	}
	return nil
}

func runIPCommand(args ...string) error {
	cmd := exec.Command("ip", args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, trimmed)
}

func isIgnorableIPCommandError(message string) bool {
	msg := strings.ToLower(message)
	return strings.Contains(msg, "cannot find device") || strings.Contains(msg, "cannot assign requested address") || strings.Contains(msg, "cannot find")
}
