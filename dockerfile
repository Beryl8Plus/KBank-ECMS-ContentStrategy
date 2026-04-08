# Build Stage
FROM golang:1.26.1-alpine3.23 AS builder

WORKDIR /app
RUN apk add --no-cache git make
COPY go.mod go.sum ./
RUN go mod download

# Install swag CLI for documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

# Generate Swagger Documentation
RUN swag init -g cmd/server/main.go --output docs/swagger --parseDependency --parseInternal

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kbank-ecms ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate/

# Run Stage
FROM alpine:3.23
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/kbank-ecms .
COPY --from=builder /app/migrate .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs/swagger ./docs/swagger
EXPOSE 8081

CMD ["sh", "-c", "./migrate && ./kbank-ecms"]
