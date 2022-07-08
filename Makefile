BUILD := $(abspath ./bin)

.PHONY: setup
setup:
	mkdir -p $(BUILD)

.PHONY: stringer
stringer: setup
	go build -o bin/stringer golang.org/x/tools/cmd/stringer

.PHONY: generate
generate: stringer
	go generate ./...

.PHONY: test
test:
	go test ./...

all: generate test
