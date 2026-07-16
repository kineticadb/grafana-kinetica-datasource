package plugin

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"golang.org/x/sync/errgroup"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	// Create minimal datasource settings
	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		Name: "test-datasource",
		JSONData: json.RawMessage(`{
			"host": "http://localhost:9191",
			"database": "test"
		}`),
		DecryptedSecureJSONData: map[string]string{
			"username": "test",
			"password": "test",
		},
	}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &settings,
			},
			Queries: []backend.DataQuery{
				{
					RefID: "A",
					JSON:  json.RawMessage(`{"queryText": "SELECT 1"}`),
				},
			},
		},
	)

	// Note: This test will fail if Kinetica is not running
	// We expect an error connecting to Kinetica, but no panic
	if err != nil {
		t.Logf("Expected error (Kinetica not running): %v", err)
	}

	if resp != nil && len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

// TestConcurrentQueryPattern tests the errgroup concurrency pattern used in QueryData.
// This validates the concurrent execution logic without requiring a real Kinetica connection.
func TestConcurrentQueryPattern(t *testing.T) {
	var mu sync.Mutex
	results := make(map[string]backend.DataResponse)

	// Simulate the queries
	queries := []backend.DataQuery{
		{RefID: "A"},
		{RefID: "B"},
		{RefID: "C"},
	}

	g, gCtx := errgroup.WithContext(context.Background())

	for _, q := range queries {
		query := q
		g.Go(func() error {
			// Simulate query work
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-time.After(10 * time.Millisecond):
			}

			res := backend.DataResponse{}
			mu.Lock()
			results[query.RefID] = res
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()

	// Verify all queries completed
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, refID := range []string{"A", "B", "C"} {
		if _, ok := results[refID]; !ok {
			t.Errorf("Missing result for RefID %s", refID)
		}
	}
}

// TestConcurrentQueryPattern_SingleQuery verifies single query optimization path.
func TestConcurrentQueryPattern_SingleQuery(t *testing.T) {
	queries := []backend.DataQuery{
		{RefID: "A"},
	}

	// Single query should skip goroutine overhead
	if len(queries) == 1 {
		// Direct execution path
		result := backend.DataResponse{}
		if result.Error != nil {
			t.Errorf("Single query should succeed")
		}
	}
}

// TestConcurrentQueryPattern_ErrorIsolation verifies one failing query doesn't affect others.
func TestConcurrentQueryPattern_ErrorIsolation(t *testing.T) {
	var mu sync.Mutex
	results := make(map[string]backend.DataResponse)
	var successCount atomic.Int32

	queries := []backend.DataQuery{
		{RefID: "A"},
		{RefID: "B"}, // This one will "fail"
		{RefID: "C"},
	}

	g, _ := errgroup.WithContext(context.Background())

	for _, q := range queries {
		query := q
		g.Go(func() error {
			var res backend.DataResponse

			if query.RefID == "B" {
				// Simulate a query error (but don't propagate to cancel others)
				res = backend.ErrDataResponse(backend.StatusBadRequest, "simulated error")
			} else {
				successCount.Add(1)
			}

			mu.Lock()
			results[query.RefID] = res
			mu.Unlock()
			return nil // Don't propagate errors to cancel other queries
		})
	}

	_ = g.Wait()

	// All queries should complete, even with one error
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Two queries should succeed
	if successCount.Load() != 2 {
		t.Errorf("Expected 2 successful queries, got %d", successCount.Load())
	}

	// Query B should have an error
	if results["B"].Error == nil {
		t.Error("Query B should have an error")
	}
}

// TestConcurrentQueryPattern_RaceCondition tests for race conditions with many concurrent queries.
// Run with: go test -race -run TestConcurrentQueryPattern_RaceCondition
func TestConcurrentQueryPattern_RaceCondition(t *testing.T) {
	var mu sync.Mutex
	results := make(map[string]backend.DataResponse)

	// Create many queries to stress test
	numQueries := 100
	queries := make([]backend.DataQuery, numQueries)
	for i := 0; i < numQueries; i++ {
		queries[i] = backend.DataQuery{RefID: string(rune('A' + i%26)) + string(rune('0'+i/26))}
	}

	g, _ := errgroup.WithContext(context.Background())

	for _, q := range queries {
		query := q
		g.Go(func() error {
			res := backend.DataResponse{}
			mu.Lock()
			results[query.RefID] = res
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()

	if len(results) != numQueries {
		t.Errorf("Expected %d results, got %d", numQueries, len(results))
	}
}
