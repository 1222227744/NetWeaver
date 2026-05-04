package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"netweaver-backend/pkg/config"
	"netweaver-backend/pkg/protocol"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = config.DefaultControllerURL
	}

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) Ping(ctx context.Context) (protocol.PingResponse, error) {
	var result protocol.PingResponse
	if err := c.getJSON(ctx, "/", &result); err != nil {
		return protocol.PingResponse{}, err
	}
	return result, nil
}

func (c *Client) RegisterNode(ctx context.Context, req protocol.RegisterNodeRequest) (protocol.NodeInfo, error) {
	var result protocol.NodeInfo
	if err := c.postJSON(ctx, "/nodes", req, &result); err != nil {
		return protocol.NodeInfo{}, err
	}
	return result, nil
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
