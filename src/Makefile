all: bin/gprompt bin/geval

bin/gprompt: gprompt.go $(wildcard *//*.go)
	@mkdir -p bin
	go build -o bin/gprompt .

bin/geval: cmd/geval/main.go $(wildcard *//*.go)
	@mkdir -p bin
	go build -o bin/geval ./cmd/geval

clean:
	rm -rf bin/

.PHONY: all clean
