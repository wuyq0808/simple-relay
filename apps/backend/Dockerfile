# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared module first
COPY apps/shared ./apps/shared

# Copy service files
COPY apps/backend/go.mod apps/backend/go.sum ./apps/backend/
WORKDIR /app/apps/backend
RUN go mod download

# Copy source code
COPY apps/backend ./
WORKDIR /app/apps/backend

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/apps/backend/main .

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]