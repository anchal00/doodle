.PHONY: drop-db init-db gen-mocks test

build:
	go build -o bin/ ./cmd/doodle

mocks:
	go generate ./...

lint:
	golangci-lint run

test:
	go test -v ./...

init-db:
	touch doodle.db

drop-db:
	rm doodle.db

