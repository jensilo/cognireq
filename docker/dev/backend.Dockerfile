# syntax=docker/dockerfile:1
FROM golang:1.24

# Install tools
RUN go install github.com/air-verse/air@v1.61.7 && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0

# Create cache directories and set ownership to UID 1000
RUN mkdir -p /go/cache /go/pkg /.cache && \
    chown -R 1000:1000 /go/cache /go/pkg /.cache

WORKDIR /app

COPY ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]