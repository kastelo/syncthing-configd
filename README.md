# Syncthing-Configd

A daemon that automatically manages certain aspects of the Syncthing
configuration.

## Principle of operation

Syncthing-Configd is a sidecar process to Syncthing itself. It listens to
events, processes them according to a configured ruleset, and applies
configuration changes to Syncthing accordingly.

## Usage

### Configuration

A configuration file contains the Syncthing instance(s) to connect to,
patterns to apply to incoming device connections, and other directives.
Initially, one or more `syncthing` sections defines the Syncthing
instance(s) to connect to:

```
syncthing {
    address: "127.0.0.1:8081"
    api_key: "abc123"
}
syncthing {
    address: "127.0.0.1:8082"
    api_key: "abc123"
}
```

### Adding devices and folders

The daemon listens to Syncthing events informing it of of "rejected
devices". A device is rejected when an incoming connection is received but
Syncthing is not configured to accept a connection from that device.

For each rejected device it checks the configured patterns to see if source
address matches a known network. If so, it applies the pattern to add the
device and share the relates set of folders with it. Folders can be created
if they don't exist beforehand, with a pattern for the ID and path.

Patterns are defined in the configuration file:

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

    # Also an individual folder for each device. The `settings` are used if
    # the folder needs to be created, otherwise the device is simply added
    # to the existing folder configuration.
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
definition](https://github.com/kastelo/syncthing-configd/blob/main/proto/config.proto).
In general it closely mirrors the config options of Syncthing itself.

### Garbage collecting unused device & folders

Devices and folders that are no longer in use can be automatically removed.
Devices are removed if they have not been "seen" (i.e., had an active
connection) for a given number of days. Folders are removed if they are not
shared with any other device -- typically, after a device has been removed.

```
garbage_collect {
    run_every_s: 86400  # once a day, midnight UTC
    unseen_devices_days: 90
    unshared_folders: true
}
```

## Installation

### Debian Package

A Debian package is available as the artifact of the latest [build
run](https://github.com/kastelo/syncthing-configd/actions/workflows/build.yml).

### Docker Image

It's easiest to use the precompiled Docker image. Assuming a configuration file in
`/etc/syncthing-configd/configd.conf`, something like the command below
will start syncthing-configd.

```
% docker run -d --restart always --name syncthing-configd \
    -v /etc/syncthing-configd/configd.conf:/etc/syncthing-configd/configd.conf \
    ghcr.io/kastelo/syncthing-configd:latest
```

Note that since a Docker container runs in a separate namespace it will not
be able to access a Syncthing instance on the same machine by using the
`127.0.0.1` address.

---

Copyright &copy; Kastelo AB. Licensed under the Mozilla Public License
Version 2.0.