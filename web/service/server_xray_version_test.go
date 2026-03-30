package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestXrayCoreVersion_StripsVPrefix(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v26.2.6"}`))
	}))
	defer server.Close()

	version, err := fetchLatestXrayCoreVersion(http.DefaultClient, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "26.2.6" {
		t.Fatalf("expected version 26.2.6, got %q", version)
	}
}
