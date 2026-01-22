package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("uses default URL when empty", func(t *testing.T) {
		client := NewClient("")
		if client.baseURL != DefaultBaseURL {
			t.Errorf("expected %s, got %s", DefaultBaseURL, client.baseURL)
		}
	})

	t.Run("uses custom URL when provided", func(t *testing.T) {
		customURL := "http://custom:1234"
		client := NewClient(customURL)
		if client.baseURL != customURL {
			t.Errorf("expected %s, got %s", customURL, client.baseURL)
		}
	})
}

func TestIsRunning(t *testing.T) {
	t.Run("returns true when server responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ollama is running"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		if !client.IsRunning() {
			t.Error("expected IsRunning to return true")
		}
	})

	t.Run("returns false when server is unavailable", func(t *testing.T) {
		client := NewClient("http://localhost:99999")
		if client.IsRunning() {
			t.Error("expected IsRunning to return false")
		}
	})

	t.Run("returns false when server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		if client.IsRunning() {
			t.Error("expected IsRunning to return false for 500 status")
		}
	})
}

func TestListModels(t *testing.T) {
	t.Run("returns models on success", func(t *testing.T) {
		mockModels := TagsResponse{
			Models: []Model{
				{Name: "qwen3-coder", Size: 1000000, ModifiedAt: time.Now()},
				{Name: "llama3.1:8b", Size: 2000000, ModifiedAt: time.Now()},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				t.Errorf("expected path /api/tags, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockModels)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		models, err := client.ListModels()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 2 {
			t.Errorf("expected 2 models, got %d", len(models))
		}
		if models[0].Name != "qwen3-coder" {
			t.Errorf("expected first model to be qwen3-coder, got %s", models[0].Name)
		}
	})

	t.Run("returns error when server unavailable", func(t *testing.T) {
		client := NewClient("http://localhost:99999")
		_, err := client.ListModels()
		if err == nil {
			t.Error("expected error when server unavailable")
		}
	})

	t.Run("returns error on non-OK status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.ListModels()
		if err == nil {
			t.Error("expected error on 500 status")
		}
	})
}

func TestHasModel(t *testing.T) {
	mockModels := TagsResponse{
		Models: []Model{
			{Name: "qwen3-coder", Size: 1000000},
			{Name: "llama3.1:8b", Size: 2000000},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockModels)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	t.Run("returns true for existing model", func(t *testing.T) {
		has, err := client.HasModel("qwen3-coder")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected HasModel to return true for qwen3-coder")
		}
	})

	t.Run("returns false for non-existing model", func(t *testing.T) {
		has, err := client.HasModel("nonexistent-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected HasModel to return false for nonexistent model")
		}
	})
}

func TestGetModelNames(t *testing.T) {
	mockModels := TagsResponse{
		Models: []Model{
			{Name: "qwen3-coder", Size: 1000000},
			{Name: "llama3.1:8b", Size: 2000000},
			{Name: "codestral", Size: 3000000},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockModels)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	names, err := client.GetModelNames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
	expectedNames := []string{"qwen3-coder", "llama3.1:8b", "codestral"}
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("expected name %s at index %d, got %s", expectedNames[i], i, name)
		}
	}
}
