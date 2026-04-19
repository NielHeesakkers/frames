.PHONY: build test lint fmt run changelog

# Keep the embedded changelog copy in sync with the root CHANGELOG.md.
changelog:
	cp CHANGELOG.md internal/version/changelog.md

build: changelog
	go build -tags="sqlite_fts5" -o frames ./cmd/frames

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

