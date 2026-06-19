FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o netloc8-mcp .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/netloc8-mcp .
ENTRYPOINT ["./netloc8-mcp"]
