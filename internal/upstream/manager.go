package upstream

import (
	"errors"
	"sync"
	"time"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
)

type State string

const (
	Healthy            State = "healthy"
	Degraded           State = "degraded"
	CoolingDown        State = "cooling_down"
	Disabled           State = "disabled"
	InvalidCredentials State = "invalid_credentials"
	Unreachable        State = "unreachable"
	ExhaustedQuota     State = "exhausted_quota"
)

type RuntimeUpstream struct {
	Config         config.UpstreamConfig
	State          State     `json:"state"`
	LastSuccess    time.Time `json:"last_success"`
	LastFailure    time.Time `json:"last_failure"`
	FailureCount   int       `json:"failure_count"`
	CooldownUntil  time.Time `json:"cooldown_until"`
	LastError      string    `json:"last_error"`
	LastQuotaError time.Time `json:"last_quota_error"`
}

type Manager struct {
	mu        sync.Mutex
	upstreams []*RuntimeUpstream
	rr        int
}

func NewManager(cfg config.Config) *Manager {
	return &Manager{upstreams: buildRuntimeUpstreams(cfg.Upstreams)}
}

func (m *Manager) Snapshot() []RuntimeUpstream {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RuntimeUpstream, 0, len(m.upstreams))
	for _, u := range m.upstreams {
		out = append(out, *u)
	}
	return out
}

func (m *Manager) Replace(cfgs []config.UpstreamConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstreams = buildRuntimeUpstreams(cfgs)
	if len(m.upstreams) == 0 || m.rr >= len(m.upstreams) {
		m.rr = 0
	}
}

func (m *Manager) MarkResult(id string, errType string, cooldown time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.upstreams {
		if u.Config.ID != id {
			continue
		}
		if errType == "" {
			u.LastSuccess = time.Now()
			u.FailureCount = 0
			u.State = Healthy
			u.LastError = ""
			return
		}
		u.LastFailure = time.Now()
		u.FailureCount++
		u.LastError = errType
		switch errType {
		case "quota_exhausted":
			u.State = ExhaustedQuota
			u.LastQuotaError = time.Now()
			u.CooldownUntil = time.Now().Add(cooldown)
		case "auth_invalid":
			u.State = InvalidCredentials
		case "timeout", "cloud_upstream_error", "rate_limited", "unreachable":
			u.State = CoolingDown
			u.CooldownUntil = time.Now().Add(cooldown)
		default:
			u.State = Degraded
		}
		return
	}
}

func (m *Manager) NextEligible() (*RuntimeUpstream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.upstreams) == 0 {
		return nil, errors.New("no cloud upstreams configured")
	}
	n := len(m.upstreams)
	now := time.Now()
	for i := 0; i < n; i++ {
		idx := (m.rr + i) % n
		u := m.upstreams[idx]
		if !u.Config.Enabled {
			continue
		}
		if !u.CooldownUntil.IsZero() && now.Before(u.CooldownUntil) {
			continue
		}
		m.rr = (idx + 1) % n
		return u, nil
	}
	return nil, errors.New("no healthy upstream available")
}

func (m *Manager) FindByNameOrID(s string) *RuntimeUpstream {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.upstreams {
		if u.Config.ID == s || u.Config.Name == s {
			return u
		}
	}
	return nil
}

func buildRuntimeUpstreams(cfgs []config.UpstreamConfig) []*RuntimeUpstream {
	out := make([]*RuntimeUpstream, 0, len(cfgs))
	for _, c := range cfgs {
		state := Healthy
		if !c.Enabled {
			state = Disabled
		}
		out = append(out, &RuntimeUpstream{Config: c, State: state})
	}
	return out
}
