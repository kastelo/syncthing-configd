# syncthing-autoacceptd

A daemon that automatically accepts and assigns folders to incoming clients,
based on certain criteria.

## Principle of Operation

The daemon listens to Syncthing events informing it of of "rejected
devices". A device is rejected when an incoming connection is received but
Syncthing is not configured to accept a connection from that device.
