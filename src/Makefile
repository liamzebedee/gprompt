CMDS := gprompt geval gcluster
BINS := $(addprefix bin/,$(CMDS))

all: $(BINS)

bin/%: cmd/%/main.go $(wildcard *//*.go)
	@mkdir -p bin
	go build -o $@ ./cmd/$*

clean:
	rm -rf bin/

.PHONY: all clean
