// Code-generation tools for the backend, version-locked in their own module (owner decisions D2 and
// D5, Phase 3 Group 4; decision log §9). Nothing here is imported by the API module or linked into
// its binaries. Used through `go tool -modfile=…/tools/go.mod <tool>` from go:generate directives.
//
// github.com/getkin/kin-openapi (the generator's OpenAPI parser) is raised from v0.142.0, which
// oapi-codegen v2.8.2 requires, to v0.148.0: the first release fixing GO-2026-6789 and GO-2026-6795
// (panics on crafted specifications). The generated output is byte-identical with either version.
// Drop this override when the generator itself requires a fixed version.

module github.com/hruizjr/sms/backend/tools

go 1.27

toolchain go1.27.4

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

require (
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/getkin/kin-openapi v0.148.0 // indirect; security override, see the comment at the top
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.8.2 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.24.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
