package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"proletarka_transport/internal/config"
)

const maxErrorBodyBytes = 4096

type Client struct {
	cfg        config.APIConfig
	httpClient *http.Client
	baseURL    *url.URL
}

func NewClient(cfg config.APIConfig, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("backend base url must be an absolute http/https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("backend base url must be an absolute http/https URL")
	}

	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
		baseURL:    parsed,
	}, nil
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) do(ctx context.Context, method string, path string, body any, out any) error {
	requestURL, err := c.buildURL(path)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal backend request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return fmt.Errorf("create backend request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.HeaderKey != "" {
		req.Header.Set(c.cfg.HeaderKey, c.cfg.Secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send backend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return backendStatusError(resp)
	}

	if out == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return fmt.Errorf("read backend response body: %w", err)
		}
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode backend response body: %w", err)
	}

	return nil
}

func (c *Client) buildURL(path string) (string, error) {
	parsedPath, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse backend path: %w", err)
	}
	if parsedPath.IsAbs() || parsedPath.Host != "" {
		return "", fmt.Errorf("backend path must be relative")
	}

	trimmedPath := strings.TrimLeft(parsedPath.Path, "/")
	if trimmedPath == "" {
		return "", fmt.Errorf("backend path must not be empty")
	}

	base := *c.baseURL
	base.Path = strings.TrimRight(base.Path, "/") + "/" + trimmedPath
	base.RawQuery = parsedPath.RawQuery
	base.Fragment = ""

	return base.String(), nil
}

func backendStatusError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return fmt.Errorf("backend returned status %d and response body could not be read: %w", resp.StatusCode, err)
	}

	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("backend returned status %d: %s", resp.StatusCode, bodyText)
}
