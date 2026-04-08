#!/bin/sh
podman rm -f bootcamp-pg bootcamp-upload bootcamp-web 2>/dev/null || true
podman network rm bootcamp 2>/dev/null || true
