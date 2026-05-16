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

COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/

# Run Stage
FROM alpine:latest
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/cmd/server/docs ./cmd/server/docs
EXPOSE 8082

CMD ["./server"]
