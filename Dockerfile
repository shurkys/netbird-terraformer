# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.21-alpine AS builder

WORKDIR /src

# Cache module downloads (no external deps, but keeps layers stable)
COPY go.mod ./
RUN go mod download

COPY . .

# Static build so the binary runs on any minimal base
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/netbird-importer .

# ---- Runtime stage ----
# hashicorp/terraform provides the `terraform` CLI the app shells out to at runtime
FROM hashicorp/terraform:1.14.6

# CA certificates are needed for HTTPS calls to the NetBird API and provider registry
RUN apk add --no-cache ca-certificates

WORKDIR /work

COPY --from=builder /out/netbird-importer /usr/local/bin/netbird-importer

# Default output directory used by the importer
ENV NB_MANAGEMENT_URL="https://api.netbird.io"

# Override the terraform image's default entrypoint
ENTRYPOINT ["/usr/local/bin/netbird-importer"]
CMD ["generated"]
