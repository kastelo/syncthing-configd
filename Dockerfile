FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.title "Syncthing Configuration Daemon"
LABEL org.opencontainers.image.url "https://kastelo.net/"
LABEL org.opencontainers.image.vendor "Kastelo AB"
LABEL org.opencontainers.image.source "https://github.com/kastelo/syncthing-configd"
ARG TARGETARCH
COPY bin/syncthing-configd-linux-$TARGETARCH /bin/syncthing-configd
ENTRYPOINT ["/bin/syncthing-configd"]
