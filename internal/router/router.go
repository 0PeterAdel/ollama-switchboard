package router

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
)

type Decision struct {
	Target      string
	Model       string
	RewrittenTo string
}

func Decide(cfg config.Config, path string, body []byte) Decision {
	model := extractModel(path, body)
	decision := Decision{Target: "local", Model: model, RewrittenTo: model}

	if model != "" {
		if to, ok := cfg.ModelMap[model]; ok {
			decision.RewrittenTo = to
		}
		for _, s := range cfg.Routing.CloudSuffix {
			if strings.HasSuffix(model, s) {
				decision.Target = "cloud"
				return decision
			}
		}
		for _, p := range cfg.Routing.CloudRegex {
			if matches(p, model) {
				decision.Target = "cloud"
				return decision
			}
		}
		for _, p := range cfg.Routing.LocalRegex {
			if matches(p, model) {
				decision.Target = "local"
				return decision
			}
		}
	}

	switch cfg.Routing.Policy {
	case "cloud-only", "prefer-cloud":
		decision.Target = "cloud"
	case "local-only", "prefer-local", "auto":
		decision.Target = "local"
	}
	if path == "/v1/responses" || path == "/v1/chat/completions" || path == "/v1/embeddings" {
		if cfg.Routing.Policy == "auto" && strings.Contains(model, ":cloud") {
			decision.Target = "cloud"
		}
	}
	return decision
}

func matches(pattern, value string) bool {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return rx.MatchString(value)
}

func extractModel(path string, body []byte) string {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	v, _ := m["model"].(string)
	return v
}

func RewriteModel(body []byte, to string) []byte {
	if to == "" {
		return body
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	if _, ok := m["model"]; ok {
		m["model"] = to
		b, err := json.Marshal(m)
		if err == nil {
			return b
		}
	}
	return body
}
