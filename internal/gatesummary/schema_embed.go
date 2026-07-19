package gatesummary

import (
	_ "embed"
)

// v1SchemaJSON is the embedded Draft 2020-12 v1 wire schema.
//
//go:embed schema/gate-summary-v1.schema.json
var v1SchemaJSON []byte

// v2SchemaJSON is the embedded Draft 2020-12 v2 wire schema.
//
//go:embed schema/gate-summary-v2.schema.json
var v2SchemaJSON []byte

// schemaSet is the bundled, embedded Draft 2020-12 schema pair.
//
// The $id values are frozen in the schema files; they are the public
// references that the validator and the schema-error translator use.
const (
	v1SchemaID = "https://leamas.local/schemas/gate-summary-v1.schema.json"
	v2SchemaID = "https://leamas.local/schemas/gate-summary-v2.schema.json"
)
