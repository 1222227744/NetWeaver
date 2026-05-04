package protocol

import "time"

type PingResponse struct {
	Msg string `json:"msg"`
}

type RegisterNodeRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type NodeInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
}
