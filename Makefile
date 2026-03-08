GOOS ?= $(shell go env GOOS)
ifeq ($(GOOS),windows)
	EXT = .exe
	RM = if exist bin rmdir /s /q bin
else
	EXT =
	RM = rm -rf bin
endif

.PHONY: build build-testserver test clean demo

build:
	go build -o bin/hotreload$(EXT) ./cmd/hotreload

build-testserver:
	go build -o bin/testserver$(EXT) ./testserver

test:
	go test -v ./...

clean:
	$(RM)

demo: build
	./bin/hotreload$(EXT) --root ./testserver --build "go build -o ./bin/testserver$(EXT) ./testserver" --exec "./bin/testserver$(EXT)"
