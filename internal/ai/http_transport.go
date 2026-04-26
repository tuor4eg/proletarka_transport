package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPTransport struct {
	client *http.Client
}

func NewHTTPTransport(client *http.Client) *HTTPTransport {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPTransport{client: client}
}

func (t *HTTPTransport) Complete(ctx context.Context, model ModelConfig, prompt Prompt) (Response, error) {
	if t == nil || t.client == nil {
		return Response{}, fmt.Errorf("http client is not configured")
	}

	endpoint, err := chatCompletionsURL(model)
	if err != nil {
		return Response{}, err
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:    model.Name,
		Messages: prompt.Messages,
	})
	if err != nil {
		return Response{}, fmt.Errorf("encode ai request: %w", err)
	}

	if model.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, model.Timeout)
		defer cancel()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("build ai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+model.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("send ai request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return Response{}, fmt.Errorf("ai provider returned status %d", httpResp.StatusCode)
	}

	var decoded chatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(httpResp.Body, 1<<20)).Decode(&decoded); err != nil {
		return Response{}, fmt.Errorf("decode ai response: %w", err)
	}

	if len(decoded.Choices) == 0 {
		return Response{}, fmt.Errorf("ai response has no choices")
	}

	return Response{
		Text:     decoded.Choices[0].Message.Content,
		ModelID:  model.ID,
		Provider: model.Provider,
	}, nil
}

func chatCompletionsURL(model ModelConfig) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(model.BaseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("AI_MODEL_%s_BASE_URL is required", model.ID)
	}

	return baseURL + "/chat/completions", nil
}

type chatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

var _ Transport = (*HTTPTransport)(nil)
