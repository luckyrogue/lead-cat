package openapi

import _ "embed"

// Spec is the embedded OpenAPI 3.1 document (source: backend/openapi/openapi.json).
//
//go:embed openapi.json
var Spec []byte
