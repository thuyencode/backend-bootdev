dev:
	air
run:
	go run .
test:
	go test -json -cover ./... | gotestfmt
fmt:
	golangci-lint fmt
lint:
	golangci-lint run
fix:
	golangci-lint run --fix
clean:
	go clean
	git clean -dfX
