.PHONY: build test lint release clean

BIN := bin/vigia

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./cmd/vigia

test:
	go test -race ./...

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

release:
	@echo "La matriz de publicación multiplataforma se añadirá en M12."

clean:
	rm -rf bin