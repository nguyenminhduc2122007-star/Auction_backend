# Backend Dockerfile — build the Go API and run it in a minimal production image.
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /auction-api ./cmd/api

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /auction-api /usr/local/bin/auction-api

USER app
EXPOSE 8080

CMD ["/usr/local/bin/auction-api"]
