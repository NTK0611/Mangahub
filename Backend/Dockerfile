# Stage 1: Build
FROM golang:1.26-bookworm AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct

# CGO needs gcc for go-sqlite3
RUN apt-get update && apt-get install -y gcc musl-dev && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/server ./cmd/${SERVICE_NAME}

# Stage 2: Runtime
FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    tzdata \
    ca-certificates \
    libc6 \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app/data

COPY --from=builder /app/bin/server .

CMD ["./server"]