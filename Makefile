.PHONY: gci-format lint test

gci-format:
	gci write --skip-generated -s standard -s default -s "prefix(github.com/cross402)" .

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...
