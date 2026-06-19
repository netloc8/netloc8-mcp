# Multi-stage build for the NetLoc8 MCP server.
# Stage 1: compile the Go binary.
# Stage 2: copy into a minimal Alpine image.

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /netloc8-mcp .

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /netloc8-mcp /usr/local/bin/netloc8-mcp
ENTRYPOINT ["netloc8-mcp"]
