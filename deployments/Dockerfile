# No replace, no go.work: inject helpy and infra into GOMODCACHE.
FROM golang:1.26-alpine AS builder
ENV GOMODCACHE=/go/pkg/mod
WORKDIR /app
COPY helpy /go/pkg/mod/github.com/psds-microservice/helpy@v0.1.0
COPY infra /go/pkg/mod/github.com/psds-microservice/infra@v0.0.3
COPY session-manager-service ./session-manager-service
WORKDIR /app/session-manager-service
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /session-manager-service ./cmd/session-manager-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /session-manager-service .
COPY --from=builder /app/session-manager-service/database ./database
EXPOSE 8091
CMD ["./session-manager-service", "api"]
