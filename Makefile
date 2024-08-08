version:=$(shell git describe | sed s/^v//)
ldflags:="-X kastelo.dev/syncthing-configd/internal/build.GitVersion=${version}"

.PHONY: install
install: test
	@go install -v -ldflags $(ldflags) ./cmd/syncthing-configd

.PHONY: test
test:
	@go test ./...

.PHONY: proto
proto:
	@clang-format -i proto/*.proto
	@protoc --go_out=. --go_opt module=kastelo.dev/syncthing-configd proto/*.proto
