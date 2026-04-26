package ai

import (
	"context"
	"fmt"
	"unicode/utf8"
)

type Service struct {
	enabled   bool
	router    *Router
	prompter  Prompter
	transport Transport
}

func NewService(enabled bool, router *Router, prompter Prompter, transport Transport) *Service {
	return &Service{
		enabled:   enabled,
		router:    router,
		prompter:  prompter,
		transport: transport,
	}
}

func (s *Service) Generate(ctx context.Context, req Request) (Response, error) {
	if s == nil || !s.enabled {
		return Response{}, ErrDisabled
	}
	if s.router == nil || s.prompter == nil || s.transport == nil {
		return Response{}, ErrNotConfigured
	}

	model, err := s.router.Route(req.ModelID)
	if err != nil {
		return Response{}, fmt.Errorf("route ai model: %w", err)
	}

	if model.MaxInputChars > 0 && utf8.RuneCountInString(req.Input) > model.MaxInputChars {
		return Response{}, fmt.Errorf("%w: max %d chars", ErrInputTooLong, model.MaxInputChars)
	}

	prompt, err := s.prompter.Build(req.Task, req.Input)
	if err != nil {
		return Response{}, fmt.Errorf("build ai prompt: %w", err)
	}

	resp, err := s.transport.Complete(ctx, model, prompt)
	if err != nil {
		return Response{}, fmt.Errorf("complete ai request: %w", err)
	}

	if resp.ModelID == "" {
		resp.ModelID = model.ID
	}
	if resp.Provider == "" {
		resp.Provider = model.Provider
	}

	return resp, nil
}
