# tests/

**Cross-cutting tests** that do not belong to a single Go package or frontend component.

Unit tests and most integration tests live **next to the code they test**:

- Go: `foo_test.go` beside `foo.go`
- Frontend: `Component.test.tsx` beside `Component.tsx`

## Planned sub-folders (created when needed)

| Sub-folder | Purpose |
| ---------- | ------- |
| `e2e/` | Browser end-to-end tests of the portals against a locally running stack |
| `load/` | Load and soak test scenarios; targets come from *verified* traffic requirements only |
| `provider-contract/` | A shared test suite every provider adapter must pass, run against a telco **simulator** |
| `fixtures/` | Synthetic test data — never real customer data or real phone numbers |

See [`docs/standards/testing-standards.md`](../docs/standards/testing-standards.md).

**No automated test may send a real SMS or connect to a real telco production endpoint.**
