// Package portalapi contains only code generated from the client portal API contract
// (contracts/openapi/portal-api-v1.yaml; owner decision G3-6, decision log §20) by oapi-codegen,
// plus this directive and the generator configuration in oapi-codegen.yaml. Never edit
// portalapi.gen.go by hand: change the specification or the configuration, then regenerate from
// backend/:
//
//	go generate ./internal/portalapi/...
//
// As for the public API (package publicapi), the generated code routes requests and binds
// parameters only. Authentication, CSRF checks, permission checks and request validation are done
// by the handlers in internal/httpserver.
package portalapi

//go:generate go tool -modfile=../../tools/go.mod oapi-codegen -config oapi-codegen.yaml ../../../contracts/openapi/portal-api-v1.yaml
