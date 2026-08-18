#syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache module downloads separately from source changes.
COPY go.mod ./
RUN [ -f go.sum ] && go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/node ./cmd/node

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/node /app/node

# Chain data, pending pool, and node identity are written here.
VOLUME ["/app/node_data"]

ENTRYPOINT ["/app/node"]
