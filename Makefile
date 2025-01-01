.PHONY: drop-db init-db gen-mocks test

build:
	go build -o bin/

mocks:
	go generate .

test:
	golangci-lint run && go test -v ./...

init-db:
	touch doodle.db

drop-db:
	rm doodle.db

