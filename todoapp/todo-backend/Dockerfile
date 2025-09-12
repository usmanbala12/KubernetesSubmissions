# Build stage
FROM golang:1.24.5-alpine AS builder

# Set working directory inside container
WORKDIR /app

# Copy go mod and sum files first (better cache usage)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the application (set entrypoint to cmd/api)
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./cmd/api

# Final stage
FROM alpine:latest

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /main .

# Run the application
CMD ["./main"]
