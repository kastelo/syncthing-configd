# syncthing-autoacceptd

A daemon that automatically accepts and assigns folders to incoming clients,
based on certain criteria.

## Principle of Operation

The daemon listens to Syncthing events informing it of of "rejected
devices". A device is rejected when an incoming connection is received but
Syncthing is not configured to accept a connection from that device.

For each rejected device it checks the configured patterns to see if source
address matches a known network. If so, it applies the pattern to add the
device and share the relates set of folders with it. Folders can be created
if they don't exist beforehand, with a pattern for the ID and path.

## Usage

Configuration is two-part: environment variables or flags for basic values,
and a configuration file for patterns. The basic values are as follows:

- `$SYNCTHING_AUTOACCEPTD_ADDRESS`: The API address of the Syncthing instance to manage; `127.0.0.1:8384` or similar.
- `$SYNCTHING_AUTOACCEPTD_APIKEY`: The API key which that Syncthing instance accepts.
- `$SYNCTHING_AUTOACCEPTD_PATTERNS_FILE`: The path to `patterns.conf`.

Further advanced flags may exists; `syncthing-autoacceptd --help` will list them all.

The pattern configuration is a list of patterns matching device source
addresses and the folders those devices should be assigned. An example is
worth a lot of words:

```
pattern {
    # Accept devices from 172.16.32.0/24
    accept_cidr: "172.16.32.0/24"
    accept_cidr: "127.0.0.0/8"

    settings {
        max_send_kbps: 250
        max_recv_kbps: 150
        num_connections: 1
        max_request_kib: 1024
    }

    # Assign them the folder "default".
    folder {
        id: "default"
    }

    # Also an individual folder for each device. The mode and path are used
    # if the folder needs to be created.
    folder {
        id: "${name}"
        settings {
            label: "Device ${name}'s folder"
            type: SEND_RECEIVE
            path: "/var/device-folders/${name}"
            fs_watcher_disabled: true
            rescan_interval_s: 86400
            case_sensitive_fs: true
            ignore_permissions: true
        }
    }
}

# Another pattern accepts devices from 10.0.0.0/8 and shares the "images"
# folder with them.
pattern {
    accept_cidr: "10.0.0.0/8"
    folder {
        id: "images"
    }
}
```

For the full range of settings, please see [the Protobuf
definition](https://github.com/kastelo/syncthing-autoacceptd/blob/main/proto/config.proto).
In general it closely mirrors the config options of Syncthing itself.

## Debian Package

A Debian package is available as the artifact of the latest [build
run](https://github.com/kastelo/syncthing-autoacceptd/actions/workflows/build.yml).

## Docker Image

It's easiest to use the precompiled Docker image. Assuming a pattern file in
`/etc/syncthing-autoacceptd/patterns.conf`, something like the command below
will start syncthing-autoacceptd.

```
% docker run -d --restart always --name syncthing-autoacceptd \
    -e SYNCTHING_AUTOACCEPTD_ADDRESS=192.0.2.42:8384 \
    -e SYNCTHING_AUTOACCEPTD_APIKEY=D814...6F7 \
    -e SYNCTHING_AUTOACCEPTD_PATTERNS_FILE=/etc/syncthing-autoacceptd/patterns.conf \
    -v /etc/syncthing-autoacceptd/patterns.conf:/etc/syncthing-autoacceptd/patterns.conf \
    ghcr.io/kastelo/syncthing-autoacceptd:latest
```

Note that since a Docker container runs in a separate namespace it will not
be able to access a Syncthing instance on the same machine by using the
`127.0.0.1` address.

---

Copyright &copy; Kastelo AB. Licensed under the Mozilla Public License
Version 2.0.