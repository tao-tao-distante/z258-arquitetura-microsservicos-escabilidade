# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder

ARG SERVICE
RUN case "$SERVICE" in \
      service-identity|service-task|service-notification) ;; \
      *) echo "SERVICE must be one of: service-identity, service-task, service-notification" >&2; exit 1 ;; \
    esac

WORKDIR /src
COPY shared/ ./shared/
COPY service-identity/ ./service-identity/
COPY service-task/ ./service-task/
COPY service-notification/ ./service-notification/
WORKDIR /src/${SERVICE}
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /service ./cmd/server

FROM alpine:3.20
RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=builder /service /app/service
USER appuser
EXPOSE 8080
CMD ["/app/service"]
