package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Service base URLs — configurable via environment variables.
var (
	warrenURL      = envOrDefault("WARREN_URL", "http://localhost:9090")
	chronicleURL   = envOrDefault("CHRONICLE_URL", "http://localhost:8700")
	dispatchURL    = envOrDefault("DISPATCH_URL", "http://localhost:8600")
	promptForgeURL = envOrDefault("PROMPTFORGE_URL", "http://localhost:8400")
	alexandriaURL  = envOrDefault("ALEXANDRIA_URL", "http://localhost:8500")
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// swarmClient is a shared HTTP client with reasonable timeouts for fan-out.
var swarmClient = &http.Client{
	Timeout: 3 * time.Second,
}

// SwarmOverview is the aggregated response from all backend services.
type SwarmOverview struct {
	Warren      *json.RawMessage  `json:"warren,omitempty"`
	Chronicle   *json.RawMessage  `json:"chronicle,omitempty"`
	Dispatch    *json.RawMessage  `json:"dispatch,omitempty"`
	PromptForge *json.RawMessage  `json:"promptforge,omitempty"`
	Alexandria  *json.RawMessage  `json:"alexandria,omitempty"`
	Errors      map[string]string `json:"errors"`
	FetchedAt   string            `json:"fetched_at"`
}

// fetchJSON performs a GET request and returns the raw JSON body.
func fetchJSON(ctx context.Context, url string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := swarmClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// fetchService fetches multiple endpoints from a single service and merges them
// into a single JSON object with the given keys.
func fetchService(ctx context.Context, baseURL string, endpoints map[string]string) (json.RawMessage, error) {
	result := map[string]json.RawMessage{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for key, path := range endpoints {
		wg.Add(1)
		go func(k, p string) {
			defer wg.Done()
			data, err := fetchJSON(ctx, baseURL+p)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			result[k] = data
		}(key, path)
	}
	wg.Wait()

	if len(result) == 0 && firstErr != nil {
		return nil, firstErr
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// handleSwarmOverview fans out to all services and returns a unified overview.
func (s *Server) handleSwarmOverview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	overview := SwarmOverview{
		Errors: map[string]string{},
	}

	type serviceResult struct {
		name string
		data json.RawMessage
		err  error
	}

	ch := make(chan serviceResult, 5)

	// Warren: health + agents
	go func() {
		data, err := fetchService(ctx, warrenURL, map[string]string{
			"health": "/admin/health",
			"agents": "/admin/agents",
		})
		ch <- serviceResult{"warren", data, err}
	}()

	// Chronicle: metrics + DLQ
	go func() {
		data, err := fetchService(ctx, chronicleURL, map[string]string{
			"metrics": "/api/v1/metrics/summary",
			"dlq":     "/api/v1/dlq/stats",
		})
		ch <- serviceResult{"chronicle", data, err}
	}()

	// Dispatch: stats + agents
	go func() {
		data, err := fetchService(ctx, dispatchURL, map[string]string{
			"stats":  "/api/v1/stats",
			"agents": "/api/v1/agents",
		})
		ch <- serviceResult{"dispatch", data, err}
	}()

	// PromptForge: prompt count
	go func() {
		raw, err := fetchJSON(ctx, promptForgeURL+"/api/prompts")
		if err != nil {
			ch <- serviceResult{"promptforge", nil, err}
			return
		}
		// Count array length
		var items []json.RawMessage
		if jsonErr := json.Unmarshal(raw, &items); jsonErr != nil {
			// Maybe it's an object with a count field — pass through as-is
			result, _ := json.Marshal(map[string]interface{}{"prompts": raw})
			ch <- serviceResult{"promptforge", json.RawMessage(result), nil}
			return
		}
		result, _ := json.Marshal(map[string]int{"prompt_count": len(items)})
		ch <- serviceResult{"promptforge", json.RawMessage(result), nil}
	}()

	// Alexandria: collection count
	go func() {
		raw, err := fetchJSON(ctx, alexandriaURL+"/api/collections")
		if err != nil {
			ch <- serviceResult{"alexandria", nil, err}
			return
		}
		var items []json.RawMessage
		if jsonErr := json.Unmarshal(raw, &items); jsonErr != nil {
			result, _ := json.Marshal(map[string]interface{}{"collections": raw})
			ch <- serviceResult{"alexandria", json.RawMessage(result), nil}
			return
		}
		result, _ := json.Marshal(map[string]int{"collection_count": len(items)})
		ch <- serviceResult{"alexandria", json.RawMessage(result), nil}
	}()

	// Collect results
	for i := 0; i < 5; i++ {
		res := <-ch
		if res.err != nil {
			overview.Errors[res.name] = res.err.Error()
			continue
		}
		raw := res.data
		switch res.name {
		case "warren":
			overview.Warren = &raw
		case "chronicle":
			overview.Chronicle = &raw
		case "dispatch":
			overview.Dispatch = &raw
		case "promptforge":
			overview.PromptForge = &raw
		case "alexandria":
			overview.Alexandria = &raw
		}
	}

	overview.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, overview)
}

// handleSwarmWarrenHealth proxies Warren /admin/health.
func (s *Server) handleSwarmWarrenHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	data, err := fetchJSON(ctx, warrenURL+"/admin/health")
	if err != nil {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("warren: %s", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleSwarmWarrenEvents proxies Warren /admin/events as an SSE stream.
func (s *Server) handleSwarmWarrenEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, warrenURL+"/admin/events", nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use a client without the default timeout for long-lived SSE.
	sseClient := &http.Client{Timeout: 0}
	resp, err := sseClient.Do(req)
	if err != nil {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("warren events: %s", err))
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
			}
			if readErr != nil {
				return
			}
		}
	}
}
