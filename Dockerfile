# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (if needed for go modules)
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app (replace 'main.go' if needed)
RUN go build -o main .

# Stage 2: Runtime
FROM alpine:latest  

# Set working directory in container
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port (change as needed)
EXPOSE 3000

# Command to run the executable
CMD ["./main"]
