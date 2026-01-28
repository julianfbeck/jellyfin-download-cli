package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	defaultClientName = "jellyfin-download"
	defaultVersion    = "0.1"
)

type Client struct {
	baseURL    string
	token      string
	userID     string
	deviceID   string
	deviceName string
	client     *http.Client
}

func NewClient(baseURL, token, userID, deviceID, deviceName string, timeout time.Duration) *Client {
	if deviceName == "" {
		deviceName = defaultClientName
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		userID:     userID,
		deviceID:   deviceID,
		deviceName: deviceName,
		client:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) SetAuth(token, userID string) {
	c.token = token
	c.userID = userID
}

func (c *Client) UserID() string {
	return c.userID
}

func (c *Client) AuthenticateByName(ctx context.Context, username, password string) (*AuthResponse, error) {
	payload := map[string]string{
		"Username": username,
		"Pw":       password,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/Users/AuthenticateByName", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuthHeaders(req, "")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth failed: %s", strings.TrimSpace(string(body)))
	}

	var out AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SearchItems(ctx context.Context, term string, types []string, limit int) ([]Item, error) {
	params := url.Values{}
	if term != "" {
		params.Set("SearchTerm", term)
	}
	if len(types) > 0 {
		params.Set("IncludeItemTypes", strings.Join(types, ","))
	}
	params.Set("Recursive", "true")
	if limit > 0 {
		params.Set("Limit", fmt.Sprintf("%d", limit))
	}
	if c.userID != "" {
		params.Set("UserId", c.userID)
	}

	var resp ItemsResponse
	if err := c.getJSON(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) GetItem(ctx context.Context, itemID string) (*Item, error) {
	var resp Item
	if err := c.getJSON(ctx, "/Items/"+itemID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SeriesEpisodes(ctx context.Context, seriesID string) ([]Item, error) {
	params := url.Values{}
	if c.userID != "" {
		params.Set("UserId", c.userID)
	}
	params.Set("Recursive", "true")
	params.Set("IncludeItemTypes", "Episode")
	params.Set("ParentId", seriesID)

	var resp ItemsResponse
	if err := c.getJSON(ctx, "/Items", params, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) OpenDownload(ctx context.Context, itemID string, offset int64) (*http.Response, error) {
	endpoint := fmt.Sprintf("/Items/%s/Download", itemID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	c.applyAuthHeaders(req, c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed: %s", strings.TrimSpace(string(body)))
	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, pathPart string, params url.Values, out interface{}) error {
	endpoint := c.baseURL + path.Clean("/"+strings.TrimPrefix(pathPart, "/"))
	if params != nil {
		endpoint += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	c.applyAuthHeaders(req, c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error: %s", strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) applyAuthHeaders(req *http.Request, token string) {
	auth := fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\"", defaultClientName, c.deviceName, c.deviceID, defaultVersion)
	req.Header.Set("X-Emby-Authorization", auth)
	if token != "" {
		req.Header.Set("X-Emby-Token", token)
	}
}
