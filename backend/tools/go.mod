// Code-generation tools for the backend, version-locked in their own module (owner decisions D2 and
// D5, Phase 3 Group 4; decision log §9). Nothing here is imported by the API module or linked into
// its binaries. Used through `go tool -modfile=…/tools/go.mod <tool>` from go:generate directives.
//
// github.com/getkin/kin-openapi (the generator's OpenAPI parser) is raised from v0.142.0, which
// oapi-codegen v2.8.2 requires, to v0.149.0 (owner decision, PR #8 review): the first release fixing
// all of GO-2026-6789 (fixed in v0.145.0), GO-2026-6795 (fixed in v0.148.0), and GO-2026-6800 and
// GO-2026-6785 (fixed in v0.149.0); all are crashes or panics on crafted input. The generated output
// is byte-identical with v0.142.0, v0.148.0 and v0.149.0. Drop this override when the generator itself
// requires a fixed version.
//
// sqlc v1.31.1 (Phase 4 Group 1, decision log §18) is pinned as a second tool. Four of its indirect
// dependencies are raised to their first releases fixing known advisories, so that govulncheck and the
// "Dependency review" job report nothing: golang.org/x/net v0.62.0 (GO-2026-6603, -6610, -6611, -6612,
// -6617, -6860, -6866), google.golang.org/grpc v1.83.2 (GO-2026-6061, -6348, -6443 /
// GHSA-2v4p-qf9q-27wj), github.com/google/cel-go v0.30.0 (GO-2026-6094 / GHSA-gcjh-h69q-9w9g), and
// golang.org/x/text (GO-2026-6629; v0.43.0 is what x/net v0.62.0 requires). Other golang.org/x
// modules moved with them by minimum version selection. The oapi-codegen output is byte-identical,
// and sqlc's output is checked by CI. Drop these overrides when sqlc itself requires fixed versions.

module github.com/hruizjr/sms/backend/tools

go 1.27

toolchain go1.27.4

tool (
	github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	github.com/sqlc-dev/sqlc/cmd/sqlc
)

require (
	cel.dev/expr v0.25.2 // indirect
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/cubicdaiya/gonp v1.0.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/getkin/kin-openapi v0.149.0 // indirect; security override, see the comment at the top
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/google/cel-go v0.30.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/ncruces/go-sqlite3 v0.32.0 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.8.2 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/pganalyze/pg_query_go/v6 v6.2.2 // indirect
	github.com/pingcap/errors v0.11.5-0.20250523034308-74f78ae071ee // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.0 // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20260418072757-ce92298d1124 // indirect
	github.com/riza-io/grpc-go v0.2.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.24.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/sqlc-dev/doubleclick v1.0.0 // indirect
	github.com/sqlc-dev/sqlc v1.31.1 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/wasilibs/go-pgquery v0.0.0-20250409022910-10ac41983c07 // indirect
	github.com/wasilibs/wazero-helpers v0.0.0-20240620070341-3dff1577cd52 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
