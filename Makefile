ldflags:="-X kastelo.dev/syncthing-autoacceptd/internal/build.GitVersion=$(shell git describe)"

.PHONY: install
install: test
	@go install -v -ldflags $(ldflags) ./cmd/syncthing-autoacceptd

.PHONY: test
test:
	@go test ./...

.PHONY: build
build: test build-linux-amd64 build-linux-arm64

build-linux-amd64:
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags $(ldflags) -o bin/syncthing-autoacceptd-linux-amd64 ./cmd/syncthing-autoacceptd

build-linux-arm64:
	@mkdir -p bin
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -v -ldflags $(ldflags) -o bin/syncthing-autoacceptd-linux-arm64 ./cmd/syncthing-autoacceptd

.PHONY: proto
proto:
	@clang-format -i proto/*.proto
	@protoc --go_out=. --go_opt module=kastelo.dev/syncthing-autoacceptd proto/*.proto
