#!/bin/bash

# Ensure we are in the project root
if [ ! -f "golem" ]; then
    echo "Error: golem binary not found! Please run 'go build -o golem ./cmd/golem' in the project root first."
    exit 1
fi

for tape in demo/*.tape; do
    echo "Processing $tape..."
    ./demo/setup_fixtures.sh >/dev/null 2>&1
    podman run --rm --network host -v $PWD:/vhs:z -v ./golem:/usr/bin/golem:z ghcr.io/charmbracelet/vhs "$tape"
done

echo "All GIFs generated successfully in the demo/ folder!"
