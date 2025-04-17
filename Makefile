PKG := `go list -mod=mod -f {{.Dir}} ./...`

GOPRIVATE := ""

all: lint
init: mod install-lint

install-lint:
	go install github.com/daixiang0/gci@latest
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest


mod-download:
	GOPRIVATE=$(GOPRIVATE) go mod download all

mod-tidy:
	GOPRIVATE=$(GOPRIVATE) go mod tidy -v

mod: mod-tidy mod-download

fmt:
	go fmt ./...
	gci write -s standard -s default -s "Prefix(github.com/aikts/proxyai)" -s blank -s dot $(PKG)

lint: fmt
	golangci-lint run