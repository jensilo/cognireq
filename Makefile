dev:
	sqlc generate
	go generate ./...
	go build -o ./tmp/srv ./cmd/srv/main.go