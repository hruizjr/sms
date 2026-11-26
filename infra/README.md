# infra/

Infrastructure **configuration** — not provisioned infrastructure. Nothing here is deployed: there is no
production or staging environment yet.

## Contents (each created only when an approved task needs it)

| Sub-folder | Purpose |
| ---------- | ------- |
| `docker/` | Dockerfiles. **Phase 3 Group 6:** [`backend.Dockerfile`](docker/backend.Dockerfile), the local-development backend image (`api`, `migrate`, `healthcheck`). Production images are not decided |
| `local/` | Docker Compose setup for local development ([README](local/README.md)): PostgreSQL (Group 2), plus a one-off migration service and the Go API (Group 6). The client portal runs natively. Other services only once approved |
| `observability/` | Planned, not created: Prometheus scrape configuration, alert rules, Grafana dashboards as code |

Deployment target (cloud provider, on-premises, Philippine data centre, Kubernetes or not) is an
**open decision**. See the discovery register: hosting location may be constrained by telco
connectivity (IP allowlisting, VPN/private links) and by customer or regulatory data-residency
expectations.

## Rules

- No real credentials, hostnames of production systems, or telco endpoints in this folder.
- Local development must work with **simulated** telco providers only.
- Do not add a service (Kafka, Redis, ClickHouse, ...) to local or deployment configuration until its
  ADR is accepted.
- Pin every container image by tag **and** SHA-256 digest (owner decision L2, Group 6). The local
  PostgreSQL image must stay identical to backend CI's; the "Local stack check" CI job enforces both.
- Publish local ports on `127.0.0.1` only.
