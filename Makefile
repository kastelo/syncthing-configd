version:=$(shell git describe | sed s/^v//)
ldflags:="-X kastelo.dev/syncthing-autoacceptd/internal/build.GitVersion=${version}"

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

.PHONY: debian
debian: build-linux-amd64
	@fpm -s dir -t deb \
		-p syncthing-autoacceptd-${version}-amd64.deb \
		--name syncthing-autoacceptd \
		--license mpl2 \
		--version ${version} \
		--architecture amd64 \
		--description "Syncthing Auto-Accept Daemon" \
		--url "https://syncthing.net/" \
		--maintainer "Kastelo AB <support@kastelo.net>" \
		bin/syncthing-autoacceptd-linux-amd64=/usr/sbin/syncthing-autoacceptd \
		etc/autoacceptd.conf.sample=/etc/syncthing-autoacceptd/autoacceptd.conf.sample \
		etc/syncthing-autoacceptd.service=/lib/systemd/system/syncthing-autoacceptd.service \
		etc/default-env=/etc/default/syncthing-autoacceptd \
		README.md=/usr/share/doc/syncthing-autoacceptd/README.md \
		LICENSE=/usr/share/doc/syncthing-autoacceptd/LICENSE