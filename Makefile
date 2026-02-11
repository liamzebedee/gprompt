.PHONY: all clean

all: bin/gprompt

bin/gprompt: gprompt.go src/parser/parser.go src/compiler/compiler.go src/runtime/runtime.go src/registry/registry.go
	@mkdir -p bin
	go build -o bin/gprompt .

clean:
	rm -rf bin/
