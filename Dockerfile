# ========== STAGE 1: BUILD ==========
FROM golang:1.26.2-trixie AS builder

WORKDIR /src

# 1) Dependencies first (Docker layer cache)
COPY go.mod go.sum ./
RUN go mod download

# 2) Source code
COPY . .

# 3) Static Linux binary (no CGO)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-s -w" \
	-o /bin/server \
	./cmd/server

# ========== STAGE 2: RUN ==========
FROM debian:trixie-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

RUN useradd --system --no-create-home --uid 10001 appuser

COPY --from=builder /bin/server /server
RUN chown appuser:appuser /server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/server"]
