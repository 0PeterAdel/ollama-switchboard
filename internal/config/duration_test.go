package config

import (
	"encoding/json"
	"testing"
)

func TestDurationStringUnmarshal(t *testing.T) {
	var c Config
	b := []byte(`{"retry":{"max_attempts":3,"attempt_timeout":"60s","backoff_base":"200ms","backoff_max":"2s","cooldown_duration":"30s"}}`)
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if c.Retry.AttemptTimeout.Std().Seconds() != 60 {
		t.Fatalf("unexpected duration")
	}
}
