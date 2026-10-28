# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY shared/ ./shared/
COPY service-notification/ ./service-notification/
WORKDIR /src/service-notification
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /service ./cmd/server

FROM alpine:3.20
RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=builder /service /app/service
USER appuser
EXPOSE 8080
CMD ["/app/service"]
