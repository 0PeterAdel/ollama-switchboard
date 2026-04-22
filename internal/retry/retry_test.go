package retry

import (
	"net/http"
	"testing"
)

func TestClassify429(t *testing.T) {
	typ, ok := Classify(&http.Response{StatusCode: 429}, nil)
	if typ != "rate_limited" || !ok {
		t.Fatalf("unexpected classify result")
	}
}
