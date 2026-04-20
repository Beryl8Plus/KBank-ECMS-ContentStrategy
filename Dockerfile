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
RUN swag init -g cmd/server/main.go --exclude cmd/cms-delivery,cmd/cms-runtime,cmd/migrate,internal/cms-runtime --output docs/swagger/server --parseDependency --parseInternal
RUN swag init -g cmd/cms-delivery/main.go --exclude internal/delivery/http/handler,cmd/server,cmd/cms-runtime,cmd/migrate,internal/cms-runtime --output docs/swagger/cmsdelivery --parseDependency --parseInternal
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kbank-ecms ./cmd/server/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cms-delivery ./cmd/cms-delivery/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cms-runtime ./cmd/cms-runtime/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate/

# Run Stage
FROM alpine:latest
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

COPY --from=builder /app/kbank-ecms .
COPY --from=builder /app/cms-delivery .
COPY --from=builder /app/cms-runtime .
COPY --from=builder /app/migrate .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs/swagger ./docs/swagger
EXPOSE 8081 8082 50051

CMD ["sh", "-c", "./migrate && ./kbank-ecms"]
