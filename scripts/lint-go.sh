#!/usr/bin/env sh

trap ctrl_c INT
ctrl_c() {
	exit 255
}

set -o errexit
set -o nounset
set -o xtrace

golangci-lint run --verbose
govulncheck ./...
