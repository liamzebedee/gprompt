all: build

build:
	go build -o todo .

test:
	go test -v ./...

clean:
	rm -f todo todos.json

.PHONY: all build test clean
