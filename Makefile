.PHONY: build run test clean

build:
	go build -o bin/secretary ./cmd/secretary

run:
	go run ./cmd/secretary

test:
	go test ./...

clean:
	rm -rf bin/
