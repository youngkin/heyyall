.PHONY: all
all: clean build test fmt

.PHONY: artifacts
build:
	GO111MODULE=on GOARCH=${GOARCH} go build ./heyyall.go

.PHONY: artifacts
release:
	rm -rf bin
	GOOS=windows GOARCH=amd64 go build -o bin/heyyall-windows-amd64.exe heyyall.go
	GOOS=linux GOARCH=amd64 go build -o bin/heyyall-linux-amd64 heyyall.go
	GOOS=darwin GOARCH=amd64 go build -o bin/heyyall-macos-amd64 heyyall.go
	GOOS=darwin GOARCH=arm64 go build -o bin/heyyall-macos-arm64 heyyall.go

.PHONY: test
test:
	GO111MODULE=on go test -v ./... -cover 2>&1

.PHONY: fmt
fmt:
	GO111MODULE=on go fmt ./...

.PHONY: godownload
godownload:
	go mod download

.PHONY: gotidy
gotidy:
	go mod tidy
