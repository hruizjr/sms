# infra/

Infrastructure **configuration** — not provisioned infrastructure. Nothing here is deployed in Phase 1.

## Planned contents (created only when an approved task needs them)

| Sub-folder | Purpose |
| ---------- | ------- |
| `docker/` | Dockerfiles for each backend executable and the frontend build |
| `local/` | Docker Compose setup for local development (PostgreSQL first; other services only once approved) |
| `observability/` | Prometheus scrape configuration, alert rules, Grafana dashboards as code |

Deployment target (cloud provider, on-premises, Philippine data centre, Kubernetes or not) is an
**open decision**. See the discovery register: hosting location may be constrained by telco
connectivity (IP allowlisting, VPN/private links) and by customer or regulatory data-residency
expectations.

## Rules

- No real credentials, hostnames of production systems, or telco endpoints in this folder.
- Local development must work with **simulated** telco providers only.
- Do not add a service (Kafka, Redis, ClickHouse, ...) to local or deployment configuration until its
  ADR is accepted.
