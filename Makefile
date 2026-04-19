.PHONY: build test lint fmt run

build:
	go build -o frames ./cmd/frames

test:
	go test ./... -count=1

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./frames
