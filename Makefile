version:=$(shell git describe | sed s/^v//)
ldflags:="-X kastelo.dev/syncthing-configd/internal/build.GitVersion=${version}"

.PHONY: install
install: test
	@go install -v -ldflags $(ldflags) ./cmd/syncthing-configd

.PHONY: test
test:
	@go test ./...

.PHONY: build
build: test build-linux-amd64 build-linux-arm64

build-linux-amd64:
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags $(ldflags) -o bin/syncthing-configd-linux-amd64 ./cmd/syncthing-configd

build-linux-arm64:
	@mkdir -p bin
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -v -ldflags $(ldflags) -o bin/syncthing-configd-linux-arm64 ./cmd/syncthing-configd

.PHONY: proto
proto:
	@clang-format -i proto/*.proto
	@protoc --go_out=. --go_opt module=kastelo.dev/syncthing-configd proto/*.proto

.PHONY: debian
debian: build-linux-amd64
	@fpm -s dir -t deb \
		-p syncthing-configd-${version}-amd64.deb \
		--name syncthing-configd \
		--license mpl2 \
		--version ${version} \
		--architecture amd64 \
		--description "Syncthing Configuration Daemon" \
		--url "https://syncthing.net/" \
		--maintainer "Kastelo AB <support@kastelo.net>" \
		bin/syncthing-configd-linux-amd64=/usr/sbin/syncthing-configd \
		etc/configd.conf.sample=/etc/syncthing-configd/configd.conf.sample \
		etc/syncthing-configd.service=/lib/systemd/system/syncthing-configd.service \
		etc/default-env=/etc/default/syncthing-configd \
		README.md=/usr/share/doc/syncthing-configd/README.md \
		LICENSE=/usr/share/doc/syncthing-configd/LICENSE