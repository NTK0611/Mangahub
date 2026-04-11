# Stage 1: Build (Chuyển sang dùng bản chuẩn, không dùng alpine nữa)
FROM golang:1.25 AS builder

WORKDIR /app

# Dùng proxy chính thức của Google, ổn định và nhanh nhất
ENV GOPROXY=https://proxy.golang.org,direct

# Cache module Go
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build service
ARG SERVICE_NAME
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/server ./cmd/${SERVICE_NAME}

# Stage 2: Runtime (Dùng bản debian slim tương thích hoàn hảo với CGO SQLite)
FROM debian:bookworm-slim
WORKDIR /app

# Cài đặt chứng chỉ mạng và múi giờ
RUN apt-get update && apt-get install -y tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/server .

# Khởi chạy server
CMD ["./server"]