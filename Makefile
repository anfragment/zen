.PHONY: lint lint-go lint-frontend lint-frontend-fix test

lint: lint-go lint-frontend

lint-go:
	golangci-lint run

lint-frontend:
	cd frontend && npm run lint

lint-frontend-fix:
	cd frontend && npm run lint -- --fix

test:
	go test -cover ./...
