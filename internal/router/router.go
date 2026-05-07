package router

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/0PeterAdel/ollama-switchboard/internal/config"
)

const (
	TargetLocal = "local"
	TargetCloud = "cloud"
)

var regexCache sync.Map

type Decision struct {
	Target         string
	FallbackTarget string
	Model          string
	RewrittenTo    string
}

func (d Decision) Targets() []string {
	if d.FallbackTarget == "" || d.FallbackTarget == d.Target {
		return []string{d.Target}
	}
	return []string{d.Target, d.FallbackTarget}
}

func Decide(cfg config.Config, path string, body []byte) Decision {
	model := extractModel(body)
	decision := Decision{Target: TargetLocal, Model: model, RewrittenTo: model}

	if model != "" {
		if to, ok := cfg.ModelMap[model]; ok {
			decision.RewrittenTo = to
		}
		if target := explicitTarget(cfg, model); target != "" {
			decision.Target = target
			decision.FallbackTarget = fallbackForPolicy(cfg.Routing.Policy, target)
			return decision
		}
	}

	policy := strings.ToLower(cfg.Routing.Policy)
	decision.Target = primaryForPolicy(policy)
	if policy == "auto" && isOpenAICompatiblePath(path) && strings.Contains(model, ":cloud") {
		decision.Target = TargetCloud
	}
	decision.FallbackTarget = fallbackForPolicy(policy, decision.Target)
	return decision
}

func explicitTarget(cfg config.Config, model string) string {
	for _, suffix := range cfg.Routing.CloudSuffix {
		if suffix != "" && strings.HasSuffix(model, suffix) {
			return TargetCloud
		}
	}
	for _, pattern := range cfg.Routing.CloudRegex {
		if matches(pattern, model) {
			return TargetCloud
		}
	}
	for _, pattern := range cfg.Routing.LocalRegex {
		if matches(pattern, model) {
			return TargetLocal
		}
	}
	return ""
}

func primaryForPolicy(policy string) string {
	switch strings.ToLower(policy) {
	case "cloud-only", "prefer-cloud":
		return TargetCloud
	default:
		return TargetLocal
	}
}

func fallbackForPolicy(policy, target string) string {
	switch strings.ToLower(policy) {
	case "prefer-local", "prefer-cloud":
		if target == TargetLocal {
			return TargetCloud
		}
		return TargetLocal
	default:
		return ""
	}
}

func matches(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp).MatchString(value)
	}
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	actual, _ := regexCache.LoadOrStore(pattern, rx)
	return actual.(*regexp.Regexp).MatchString(value)
}

func isOpenAICompatiblePath(path string) bool {
	return path == "/v1/responses" || path == "/v1/chat/completions" || path == "/v1/embeddings"
}

func extractModel(body []byte) string {
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
