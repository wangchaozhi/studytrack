FROM golang:1.26.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /studytrack-api ./cmd/server

FROM alpine:3.22
RUN adduser -D -u 10001 appuser
USER appuser
COPY --from=builder /studytrack-api /usr/local/bin/studytrack-api
EXPOSE 8080
ENTRYPOINT ["studytrack-api"]
