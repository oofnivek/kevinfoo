.PHONY: run build css css-watch dev test vet tidy clean

PORT ?= 8080

# Run the application (builds CSS once first), killing anything already on PORT
run: css
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || true
	go run ./cmd/server

# Build the production binary (builds minified CSS first)
build: css-prod
	go build -o bin/server ./cmd/server

# Build Tailwind CSS once
css:
	npm run css:build

# Build minified Tailwind CSS for production
css-prod:
	npm run css:prod

# Rebuild Tailwind CSS on change
css-watch:
	npm run css:watch

# Run tests
test:
	go test ./...

# Run go vet
vet:
	go vet ./...

# Tidy go.mod
tidy:
	go mod tidy

# Remove build artifacts
clean:
	@rm -f bin/server web/static/css/output.css coverage.out
