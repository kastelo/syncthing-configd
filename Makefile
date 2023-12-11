ldflags:="-X kastelo.dev/syncthing-autoacceptd/internal/build.GitVersion=$(shell git describe)"

.PHONY: build
build: proto
	@mkdir -p bin
	@go build -v -ldflags $(ldflags) -o bin/syncthing-autoacceptd ./cmd/autoacceptd

.PHONY: proto
proto:
	@clang-format -i proto/*.proto
	@protoc --go_out=. --go_opt module=kastelo.dev/syncthing-autoacceptd proto/*.proto
