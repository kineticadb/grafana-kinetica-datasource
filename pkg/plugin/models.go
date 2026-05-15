package plugin

import "encoding/json"

type DataSourceSettings struct {
	URL           string `json:"url"`
	Username      string `json:"username"`
	TlsSkipVerify bool   `json:"tlsSkipVerify"`
}

// KineticaSelect represents a single column selection from the frontend builder
type KineticaSelect struct {
	Column    string `json:"column"`
	Aggregate string `json:"aggregate"`
	Alias     string `json:"alias"`
}

type QueryModel struct {
	RawSql  string `json:"rawSql"`
	Builder struct {
		Schema  string           `json:"schema"`
		Table   string           `json:"table"`
		Selects []KineticaSelect `json:"selects"`
	} `json:"builder"`
}

// ---------------------------------------------------------
// Structs for Parsing Kinetica Table Metadata
// ---------------------------------------------------------

type KineticaTypeDefinition struct {
	Type   string           `json:"type"`
	Name   string           `json:"name"`
	Fields []KineticaColumn `json:"fields"`
}

type KineticaColumn struct {
	Name       string   `json:"name"`
	Type       any      `json:"type"`
	Properties []string `json:"column_properties"`
}

// ---------------------------------------------------------

type KineticaRawResponse struct {
	ColumnHeaders   []string `json:"column_headers"`
	ColumnDatatypes []string `json:"column_datatypes"`
	RawData         map[string]json.RawMessage
}

func (k *KineticaRawResponse) UnmarshalJSON(b []byte) error {
	type Alias KineticaRawResponse
	aux := &struct{ *Alias }{Alias: (*Alias)(k)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	if err := json.Unmarshal(b, &k.RawData); err != nil {
		return err
	}
	return nil
}
