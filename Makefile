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

.PHONY: web
web:
	cd web && (pnpm install || npm install) && (pnpm build || npm run build)
	rm -rf internal/frontend/dist
	cp -r web/build internal/frontend/dist

