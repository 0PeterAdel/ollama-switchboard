package retry

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

type Policy struct {
	MaxAttempts int
	Base        time.Duration
	Max         time.Duration
}

func Backoff(p Policy, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	d := p.Base * (1 << (attempt - 1))
	if d > p.Max {
		d = p.Max
	}
	j := time.Duration(rand.Int63n(int64(d / 4)))
	return d + j
}

func Classify(resp *http.Response, err error) (typ string, retriable bool) {
	if err != nil {
		if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "timeout") {
			return "timeout", true
		}
		return "unreachable", true
	}
	if resp == nil {
		return "cloud_upstream_error", true
	}
	switch resp.StatusCode {
	case 400, 404, 422:
		return "client_error", false
	case 401, 403:
		return "auth_invalid", true
	case 429:
		return "rate_limited", true
	case 500, 502, 503, 504:
		return "cloud_upstream_error", true
	default:
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return "", false
		}
		return "cloud_upstream_error", false
	}
}
