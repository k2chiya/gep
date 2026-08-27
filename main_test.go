package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadPDF(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/S100TEST" || r.URL.Query().Get("type") != "2" || r.URL.Query().Get("Subscription-Key") != "secret" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("%PDF-1.7\ntest")),
			Header:     make(http.Header),
		}, nil
	})}

	path := filepath.Join(t.TempDir(), "S100TEST.pdf")
	if err := download(context.Background(), client, "https://example.test", "S100TEST", 2, "secret", path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "%PDF-1.7\ntest" {
		t.Fatalf("unexpected file: %q", got)
	}
}

func TestDownloadDoesNotLeaveFileForAPIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"message":"not found"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	path := filepath.Join(t.TempDir(), "S100TEST.zip")
	err := download(context.Background(), client, "https://example.test", "S100TEST", 1, "secret", path)
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("output should not exist; stat error: %v", statErr)
	}
}
