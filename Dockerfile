# Build stage for SQLC generation
FROM golang:1.23-alpine AS sqlc-builder

# Install SQLC
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Set working directory
WORKDIR /app

# Copy go mod files for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy SQLC configuration and SQL files
COPY sqlc.yaml ./
COPY internal/db/migrations/ ./internal/db/migrations/
COPY internal/auth/users.sql ./internal/auth/
COPY internal/org/queries.sql ./internal/org/
COPY internal/org/org_members.sql ./internal/org/
COPY internal/org/org_members_extended.sql ./internal/org/
COPY internal/secretgroup/queries.sql ./internal/secretgroup/
COPY internal/secretgroup/secret_group_members.sql ./internal/secretgroup/
COPY internal/secretgroup/secret_group_members_extended.sql ./internal/secretgroup/
COPY internal/environment/queries.sql ./internal/environment/
COPY internal/environment/environment_members.sql ./internal/environment/
COPY internal/environment/environment_members_extended.sql ./internal/environment/
COPY internal/iam/queries.sql ./internal/iam/
COPY internal/iam/permissions_management.sql ./internal/iam/
COPY internal/iam/enhanced_rbac_queries.sql ./internal/iam/
COPY internal/iam/ownership_transfer_queries.sql ./internal/iam/
COPY internal/groups/queries.sql ./internal/groups/
COPY internal/secret/queries.sql ./internal/secret/
COPY internal/provider/queries.sql ./internal/provider/

# Generate SQLC code
RUN sqlc generate

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy generated SQLC code from previous stage
COPY --from=sqlc-builder /app/internal/auth/gen/ ./internal/auth/gen/
COPY --from=sqlc-builder /app/internal/org/gen/ ./internal/org/gen/
COPY --from=sqlc-builder /app/internal/secretgroup/gen/ ./internal/secretgroup/gen/
COPY --from=sqlc-builder /app/internal/environment/gen/ ./internal/environment/gen/
COPY --from=sqlc-builder /app/internal/iam/gen/ ./internal/iam/gen/
COPY --from=sqlc-builder /app/internal/groups/gen/ ./internal/groups/gen/
COPY --from=sqlc-builder /app/internal/secret/gen/ ./internal/secret/gen/
COPY --from=sqlc-builder /app/internal/provider/gen/ ./internal/provider/gen/

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o kavach-backend \
    ./cmd/server

# Final stage - minimal runtime image
FROM alpine:latest

# Install curl for health checks
RUN apk add --no-cache curl

# Copy timezone data and SSL certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /app/kavach-backend /kavach-backend

# Copy model.conf file for Casbin authorization
COPY --from=builder /app/internal/authz/model.conf /internal/authz/model.conf

# Create non-root user
RUN addgroup -g 1000 appuser && adduser -D -s /bin/sh -u 1000 -G appuser appuser
USER appuser

# Expose port
EXPOSE 8080

# Health check using HTTP endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Run the application
ENTRYPOINT ["/kavach-backend"] 