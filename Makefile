.PHONY: all clean

all: bin/gprompt

bin/gprompt: gprompt.go parser/parser.go compiler/compiler.go runtime/runtime.go registry/registry.go
	@mkdir -p bin
	go build -o bin/gprompt .

clean:
	rm -rf bin/
