.PHONY: build linux test clean

BINARY := lazysvn
DIST   := dist

build:
	go build -o $(BINARY) ./cmd/lazysvn

linux:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/lazysvn

test:
	go test ./...

clean:
	rm -rf $(BINARY) $(DIST)
