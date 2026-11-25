package publicapi

// This package contains only code generated from the approved public API contract
// (contracts/openapi/public-api-v1.yaml, ADR-0015) by oapi-codegen, plus this directive and the
// generator configuration in oapi-codegen.yaml. Never edit publicapi.gen.go by hand: change the
// specification (through its own review) or the configuration, then regenerate from backend/:
//
//	go generate ./internal/publicapi/...
//
// The generator version is locked in tools/go.mod (a separate module, so the generator's own
// dependencies are not part of the API module). The command uses only the go tool and
// forward-slash relative paths, so it runs unchanged on Linux, macOS and Windows.
//
// What the generated code does and does not do: it routes requests, binds path, query and header
// parameters to typed values (reporting values that cannot be parsed into the declared type), and
// decodes JSON bodies into typed structs. It does NOT perform full OpenAPI request validation
// (patterns, lengths, enums, required body fields, additionalProperties: false, null where null is
// not allowed) and it does NOT authenticate or authorize. Both are deferred to the first endpoint
// implementation (owner decision D6, Phase 3 Group 4).

//go:generate go tool -modfile=../../tools/go.mod oapi-codegen -config oapi-codegen.yaml ../../../contracts/openapi/public-api-v1.yaml
