# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go clean -cache && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o govaguard main.go

# Final stage - distroless
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/govaguard /govaguard

# Copy embedded files (templates, static assets, and whitepapers)
COPY --from=builder /app/templates /templates
COPY --from=builder /app/static /static
COPY --from=builder /app/whitepapers /whitepapers

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/govaguard"]
