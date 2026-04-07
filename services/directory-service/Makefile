run:
	go run ./cmd/api

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

test-integration:
	go test ./internal/directory -run Integration -count=1
