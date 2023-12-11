FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.title "Syncthing Auto Accept Daemon"
LABEL org.opencontainers.image.url "https://kastelo.net/"
LABEL org.opencontainers.image.vendor "Kastelo AB"
LABEL org.opencontainers.image.source "https://github.com/kastelo/syncthing-autoacceptd"
ARG TARGETARCH
COPY bin/syncthing-autoacceptd-linux-$TARGETARCH /bin/syncthing-autoacceptd
ENTRYPOINT ["/bin/syncthing-autoacceptd"]
