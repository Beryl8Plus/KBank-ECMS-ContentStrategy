# ─────────────────────────────────────────────
# Dockerfile
# Builds: svc-contstrat-delivery
# ─────────────────────────────────────────────

# Build Stage
FROM golang:1.26-alpine AS builder
# FROM common-docker.artifactory.kasikornbank.com:8443/devops-ocp-builder/common/cert-tools:1 AS cert-tools
# FROM docker.artifactory.kasikornbank.com:8443/golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache "git~=2.52.0" "make~=4.4.1"
COPY go.mod go.sum ./
RUN go mod download

# COPY --from=cert-tools /tools/fetch-cert.sh /fetch-cert.sh

# RUN chmod +x /fetch-cert.sh
# RUN /fetch-cert.sh go.kbtg.tech 443 && update-ca-certificates

COPY cmd cmd
COPY configs configs
COPY internal internal
COPY pkg pkg
COPY go.mod go.mod
COPY go.sum go.sum

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/

# Run Stage
FROM alpine:latest
# FROM secure-based-docker.artifactory.kasikornbank.com:8443/ubi/base-ubi9:9.0

WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/cmd/server/docs ./cmd/server/docs
EXPOSE 8082

CMD ["./server"]
