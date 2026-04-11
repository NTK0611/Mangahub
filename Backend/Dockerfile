# Stage 1: Build
# Cập nhật đúng phiên bản Go 1.25.0 theo file go.mod của bạn
FROM golang:1.25.0-alpine AS builder

# Bắt buộc phải cài gcc và musl-dev để biên dịch được go-sqlite3 (CGO_ENABLED=1)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Thêm nguồn tải dự phòng để tránh lỗi mạng kết nối khi tải thư viện
ENV GOPROXY=https://goproxy.io,direct

# Cache module Go
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code
COPY . .

# Nhận tên service từ docker-compose để build đúng thư mục trong cmd/
ARG SERVICE_NAME
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/server ./cmd/${SERVICE_NAME}

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app

# Cài đặt thư viện C chuẩn cho Alpine để SQLite có thể chạy được
RUN apk add --no-cache tzdata

COPY --from=builder /app/bin/server .

# Khởi chạy server
CMD ["./server"]