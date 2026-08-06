package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBionicClient_CheckModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"local-model"}]}`))
	}))
	defer server.Close()

	client, err := NewBionicClient(Config{BaseURL: server.URL + "/v1"}, nil)
	if err != nil {
		t.Fatalf("NewBionicClient() error = %v", err)
	}
	readiness := client.Check(context.Background())
	if readiness.Status != "READY" || len(readiness.Models) != 1 {
		t.Fatalf("readiness = %+v", readiness)
	}
}

func TestBionicClient_ChatBlockedByDefault(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client, err := NewBionicClient(Config{BaseURL: server.URL + "/v1"}, nil)
	if err != nil {
		t.Fatalf("NewBionicClient() error = %v", err)
	}
	_, err = client.Chat(context.Background(), ChatRequest{
		Model:    "local-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if !errors.Is(err, ErrEgressBlocked) {
		t.Fatalf("Chat() error = %v, want ErrEgressBlocked", err)
	}
	if called {
		t.Fatal("blocked chat reached provider")
	}
}

func TestBionicClient_ChatAllowedWithExplicitFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "local-model" {
			t.Fatalf("model = %q", request.Model)
		}
		_, _ = w.Write([]byte(`{"id":"response-1","model":"local-model","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	client, err := NewBionicClient(Config{BaseURL: server.URL + "/v1", AllowChat: true}, nil)
	if err != nil {
		t.Fatalf("NewBionicClient() error = %v", err)
	}
	response, err := client.Chat(context.Background(), ChatRequest{
		Model:    "local-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if response.Choices[0].Message.Content != "ok" {
		t.Fatalf("content = %q", response.Choices[0].Message.Content)
	}
}

func TestBionicClient_RejectsNonLoopback(t *testing.T) {
	_, err := NewBionicClient(Config{BaseURL: "https://example.com/v1"}, nil)
	if err == nil {
		t.Fatal("NewBionicClient() accepted external URL")
	}
}
