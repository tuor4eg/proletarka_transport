package ai

import "fmt"

type Router struct {
	enabled        bool
	defaultModelID string
	models         map[string]ModelConfig
}

func NewRouter(enabled bool, defaultModelID string, models []ModelConfig) *Router {
	byID := make(map[string]ModelConfig, len(models))
	for _, model := range models {
		byID[model.ID] = model
	}

	return &Router{
		enabled:        enabled,
		defaultModelID: defaultModelID,
		models:         byID,
	}
}

func (r *Router) Route(modelID string) (ModelConfig, error) {
	if r == nil || !r.enabled {
		return ModelConfig{}, ErrDisabled
	}

	if modelID == "" {
		modelID = r.defaultModelID
	}

	model, ok := r.models[modelID]
	if !ok {
		return ModelConfig{}, fmt.Errorf("%w: %s", ErrUnknownModel, modelID)
	}

	return model, nil
}
