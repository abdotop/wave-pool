# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies (this will create go.sum if needed)
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]