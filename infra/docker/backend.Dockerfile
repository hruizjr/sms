# Backend image for local development (Phase 3 Group 6; owner decisions L1–L3, decision log §11).
# One image with three static binaries: api (default entrypoint), migrate, and healthcheck.
#
# Build context: backend/ (see backend/.dockerignore). Used by infra/local/compose.yaml:
#   docker compose -f infra/local/compose.yaml build
#
# Both base images are pinned by tag and by multi-platform index digest (owner decision L2). Update
# a digest only together with its tag, and record the change in the decision log.
# Not a production image: deployment is not decided (infra/README.md).

FROM golang:1.27.4-trixie@sha256:430985542100b871e108ccc00e8dcc5eef08ca14ee5825e98ebf563d8ffec4ee AS build

# Never download a different Go toolchain; build static binaries without cgo.
ENV GOTOOLCHAIN=local \
    CGO_ENABLED=0 \
    GOFLAGS=-trimpath

WORKDIR /src

# Download and verify modules first, so source changes reuse this layer.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -ldflags="-s -w -buildid=" -o /out/api ./cmd/api \
 && go build -ldflags="-s -w -buildid=" -o /out/migrate ./cmd/migrate \
 && go build -ldflags="-s -w -buildid=" -o /out/healthcheck ./cmd/healthcheck \
 && go build -ldflags="-s -w -buildid=" -o /out/dispatcher ./cmd/dispatcher \
 && go build -ldflags="-s -w -buildid=" -o /out/smsctl ./cmd/smsctl

# Runtime: distroless static (CA certificates, time zone data, /etc/passwd with "nonroot"); no shell,
# no package manager. Runs as the unprivileged user 65532.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

COPY --from=build /out/api /out/migrate /out/healthcheck /out/dispatcher /out/smsctl /usr/local/bin/

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
