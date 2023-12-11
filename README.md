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

---

Copyright &copy; Kastelo AB. Licensed under the Mozilla Public License
Version 2.0.