package tun

import (
	"fmt"
	"io"
	"log"

	"github.com/songgao/water"

	"netweaver-backend/pkg/config"
)

type Device struct {
	io.ReadWriteCloser
	Name string
}

type OpenOptions struct {
	Name      string
	Bootstrap bool
	OwnerID   int
	GroupID   int
}

func Open(options OpenOptions) (*Device, error) {
	if options.Name == "" {
		options.Name = config.DefaultTUNName
	}

	params := water.PlatformSpecificParams{
		Name: options.Name,
	}
	if options.Bootstrap {
		targetUID, targetGID := defaultTargetOwnerGroup()
		if options.OwnerID >= 0 {
			targetUID = options.OwnerID
		}
		if options.GroupID >= 0 {
			targetGID = options.GroupID
		}

		params.Persist = true
		params.Permissions = &water.DevicePermissions{
			Owner: uint(targetUID),
			Group: uint(targetGID),
		}
		log.Printf("bootstrap mode: persist=%t owner=%d group=%d ifname=%s", params.Persist, targetUID, targetGID, options.Name)
	}

	cfg := water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: params,
	}

	ifce, err := water.New(cfg)
	if err == nil {
		return &Device{
			ReadWriteCloser: ifce,
			Name:            ifce.Name(),
		}, nil
	}

	if !options.Bootstrap && isPermissionErr(err) {
		f, name, directErr := openTunDirect(options.Name)
		if directErr == nil {
			log.Printf("water.New hit permission error (%v), fallback direct-open succeeded", err)
			return &Device{
				ReadWriteCloser: f,
				Name:            name,
			}, nil
		}
		return nil, fmt.Errorf("create/open TUN failed: water=%v, direct=%v\n%s", err, directErr, tunCreateHints(err, options.Name))
	}

	return nil, fmt.Errorf("create/open TUN failed: %w\n%s", err, tunCreateHints(err, options.Name))
}
