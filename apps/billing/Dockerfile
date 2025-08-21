FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared module first
COPY apps/shared ./apps/shared

# Copy service files
COPY apps/billing/go.mod apps/billing/go.sum ./apps/billing/
WORKDIR /app/apps/billing
RUN go mod download

# Copy source code
COPY apps/billing ./
WORKDIR /app/apps/billing

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary
COPY --from=builder /app/apps/billing/main .

# Expose port
EXPOSE 8081

# Run the binary
CMD ["./main"]