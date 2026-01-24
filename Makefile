.PHONY: build install clean test

build:
	go build -o lazygit-lite ./cmd/lazygit-lite

install:
	go install ./cmd/lazygit-lite

clean:
	rm -f lazygit-lite

test:
	go test ./...

run:
	go run ./cmd/lazygit-lite
