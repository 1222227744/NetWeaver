package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	PSK        string
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = config.DefaultControllerURL
	}

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: config.DefaultRequestTimeout,
		},
		PSK: config.NodePSK(),
	}
}

func NewWithPSK(baseURL string, psk string) *Client {
	client := New(baseURL)
	client.PSK = strings.TrimSpace(psk)
	return client
}

func (c *Client) Ping(ctx context.Context) (protocol.PingResponse, error) {
	var result protocol.PingResponse
	if err := c.getJSON(ctx, "/", &result); err != nil {
		return protocol.PingResponse{}, err
	}
	return result, nil
}

func (c *Client) RegisterNode(ctx context.Context, req protocol.RegisterNodeRequest) (protocol.RegisterNodeResponse, error) {
	var result protocol.APIResponse[protocol.RegisterNodeResponse]
	if err := c.postJSON(ctx, "/api/v1/nodes/register", req, &result); err != nil {
		return protocol.RegisterNodeResponse{}, err
	}
	if err := ensureAPIResponse(result.Code, result.Msg); err != nil {
		return protocol.RegisterNodeResponse{}, err
	}
	return result.Data, nil
}

func (c *Client) Heartbeat(ctx context.Context, nodeID string, req protocol.HeartbeatRequest) (protocol.HeartbeatResponse, error) {
	var result protocol.APIResponse[protocol.HeartbeatResponse]
	path := "/api/v1/nodes/" + url.PathEscape(nodeID) + "/heartbeat"
	if err := c.postJSON(ctx, path, req, &result); err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	if err := ensureAPIResponse(result.Code, result.Msg); err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	return result.Data, nil
}

func (c *Client) GetPeers(ctx context.Context, nodeID string) (protocol.PeersResponse, error) {
	var result protocol.APIResponse[protocol.PeersResponse]
	path := "/api/v1/nodes/" + url.PathEscape(nodeID) + "/peers"
	if err := c.getJSON(ctx, path, &result); err != nil {
		return protocol.PeersResponse{}, err
	}
	if err := ensureAPIResponse(result.Code, result.Msg); err != nil {
		return protocol.PeersResponse{}, err
	}
	return result.Data, nil
}

func (c *Client) GetNodeMetrics(ctx context.Context, nodeID string, targetID string, timeRange string) (protocol.NodeMetricsResponse, error) {
	var result protocol.APIResponse[protocol.NodeMetricsResponse]
	if timeRange == "" {
		timeRange = "1h"
	}

	query := url.Values{}
	query.Set("target_id", targetID)
	query.Set("time_range", timeRange)
	path := "/api/v1/nodes/" + url.PathEscape(nodeID) + "/metrics?" + query.Encode()
	if err := c.getJSON(ctx, path, &result); err != nil {
		return protocol.NodeMetricsResponse{}, err
	}
	if err := ensureAPIResponse(result.Code, result.Msg); err != nil {
		return protocol.NodeMetricsResponse{}, err
	}
	return result.Data, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, out)
}

func (c *Client) postJSON(ctx context.Context, path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doJSON(req, out)
}

func (c *Client) doJSON(req *http.Request, out any) error {
	if c.PSK != "" && strings.HasPrefix(req.URL.Path, "/api/v1/nodes") {
		req.Header.Set("Authorization", "Bearer "+c.PSK)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("controller returned %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func ensureAPIResponse(code int, msg string) error {
	if code == protocol.CodeSuccess {
		return nil
	}
	if msg == "" {
		msg = "controller returned an unsuccessful API response"
	}
	return fmt.Errorf("%s (code %d)", msg, code)
}
