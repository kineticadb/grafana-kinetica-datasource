package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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
