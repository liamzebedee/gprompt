all: bin/gprompt

bin/gprompt: gprompt.go $(wildcard *//*.go)
	@mkdir -p bin
	go build -o bin/gprompt .

clean:
	rm -rf bin/

.PHONY: all clean
