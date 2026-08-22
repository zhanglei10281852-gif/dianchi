test:
	GOTOOLCHAIN=local go test ./... -count=1
race:
	GOTOOLCHAIN=local go test -race ./... -count=1
build:
	GOTOOLCHAIN=local go build ./...
vet:
	GOTOOLCHAIN=local go vet ./...
