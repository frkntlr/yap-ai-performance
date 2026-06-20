.PHONY: build-all build-linux build-darwin-amd64 build-darwin-arm64 build-windows test vet clean

build-all: build-linux build-darwin-amd64 build-darwin-arm64 build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/yap-linux-amd64 ./cmd/yap

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/yap-darwin-amd64 ./cmd/yap

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/yap-darwin-arm64 ./cmd/yap

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/yap-windows-amd64.exe ./cmd/yap

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf dist/
