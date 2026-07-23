package gatesummary

import (
	"github.com/s1onique/leamas/internal/gatesummary/schema"
)

// The schema authority is owned by the internal/gatesummary/schema
// subpackage. The thin re-export below keeps the existing compile and
// validation paths stable while routing every read through the
// canonical subpackage. The IDs are the stable URN identifiers
// declared in the schema subpackage; they are not network-fetch
// requirements.
var (
	v1SchemaJSON = schema.MustBytes(schema.VersionV1)
	v2SchemaJSON = schema.MustBytes(schema.VersionV2)

	v1SchemaID = schema.SchemaIDV1
	v2SchemaID = schema.SchemaIDV2
)
