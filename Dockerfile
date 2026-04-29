# ─────────────────────────────────────────────
# Dockerfile
# Builds: svc-contstrat-delivery
# ─────────────────────────────────────────────

# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git make
COPY go.mod go.sum ./
RUN go mod download

# Install swag CLI for documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

# Generate Swagger Documentation (delivery only)
RUN swag init -g cmd/server/main.go \
    --output docs/swagger/server \
    --packageName svc_contstrat_delivery \
    --parseDependency --parseInternal

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/

# Run Stage
FROM alpine:latest
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs/swagger/server ./docs/swagger/server
EXPOSE 8082

CMD ["./server"]
