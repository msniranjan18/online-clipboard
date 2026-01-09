# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code (pkg, cmd, static)
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
# Use /app to match your local structure
WORKDIR /app

COPY --from=builder /app/main .
# This copies static into /app/static
COPY --from=builder /app/static ./static

EXPOSE 8080
# Run the binary from /app
CMD ["./main"]
