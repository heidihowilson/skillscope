.PHONY: all build test demo install clean

all: build

build:
	go build -o bin/skillscope ./cmd/skillscope

test:
	go test ./...

install:
	go install ./cmd/skillscope

# Boot the TUI pointed at testdata/.
demo: build
	./bin/skillscope --demo-home $(CURDIR)/testdata/home --demo-repo $(CURDIR)/testdata/project

clean:
	rm -rf bin/
