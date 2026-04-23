# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git make
COPY go.mod go.sum ./
RUN go mod download

# Install swag CLI for documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

# Generate Swagger Documentation
RUN swag init -g cmd/svc-contstrat-backoffice/main.go \
    --output docs/swagger/svc-contstrat-backoffice \
    --packageName svc_contstrat_backoffice \
    --parseDependency --parseInternal \
    --exclude cmd/svc-contstrat-delivery,cmd/migrate
RUN swag init -g cmd/svc-contstrat-delivery/main.go \
    --output docs/swagger/svc-contstrat-delivery \
    --packageName svc_contstrat_delivery \
    --parseDependency --parseInternal \
    --exclude cmd/svc-contstrat-backoffice,cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o svc-contstrat-backoffice ./cmd/svc-contstrat-backoffice/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o svc-contstrat-delivery ./cmd/svc-contstrat-delivery/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate/

# Run Stage
FROM alpine:latest
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/svc-contstrat-backoffice .
COPY --from=builder /app/svc-contstrat-delivery .
COPY --from=builder /app/migrate .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs/swagger ./docs/swagger
EXPOSE 8081 8082 50051

CMD ["sh", "-c", "./migrate && ./svc-contstrat-backoffice"]
