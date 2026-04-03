# Build Stage
FROM golang:1.25.5-alpine3.23 AS builder

WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kbank-ecms main.go

# Run Stage
FROM alpine:3.23
WORKDIR /app/
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok
COPY --from=builder /app/kbank-ecms .
EXPOSE 8081

CMD ["./kbank-ecms"]
