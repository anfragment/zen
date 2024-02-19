.PHONY: run
run:
	wails dev

.PHONY: lint
lint: lint-go lint-frontend

.PHONY: lint-go
lint-go:
	golangci-lint run

.PHONY: lint-frontend
lint-frontend:
	cd frontend && npm run lint

.PHONY: lint-frontend-fix
lint-frontend-fix:
	cd frontend && npm run lint -- --fix

.PHONY: test
test:
	go test -cover ./...
