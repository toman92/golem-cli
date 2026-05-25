#!/bin/bash

# Define paths
FIXTURES_DIR="demo/fixtures"
PROJECTS_DIR="$FIXTURES_DIR/projects"

# Clean up old fixtures
echo "Cleaning up old fixtures..."
rm -rf "$FIXTURES_DIR"

# Create base directories
mkdir -p "$PROJECTS_DIR/golem-core/docs"
mkdir -p "$PROJECTS_DIR/web-frontend/src"
mkdir -p "$PROJECTS_DIR/build/logs"
mkdir -p "$FIXTURES_DIR/projects-dump"

# Populate golem-core
echo "# Golem Core" > "$PROJECTS_DIR/golem-core/README.md"
echo "package main" > "$PROJECTS_DIR/golem-core/main.go"
echo "GPLv3 License" > "$PROJECTS_DIR/golem-core/LICENSE.md"
echo "API docs go here" > "$PROJECTS_DIR/golem-core/docs/api.txt"
echo "# Architecture" > "$PROJECTS_DIR/golem-core/docs/ARCHITECTURE.md"

# Populate web-frontend
echo "# Web Frontend" > "$PROJECTS_DIR/web-frontend/README.md"
echo "console.log('hello');" > "$PROJECTS_DIR/web-frontend/src/app.js"
echo "Please contribute!" > "$PROJECTS_DIR/web-frontend/CONTRIBUTING.md"
echo "PORT=8080" > "$PROJECTS_DIR/web-frontend/config.txt"

# Populate build
echo "Error: Out of memory" > "$PROJECTS_DIR/build/logs/error.log"
echo "System initialized" > "$PROJECTS_DIR/build/logs/system.txt"
echo "# DEPRECATED" > "$PROJECTS_DIR/build/README.md"

# Dummy file for projects-dump to ensure it exists for fuzzy match typos
echo "Empty" > "$FIXTURES_DIR/projects-dump/empty.txt"

# Set up external environment for deep search and sandbox escapes
EXTERNAL_ENV="$HOME/golem_test_env"
echo "Setting up external test environment in $EXTERNAL_ENV..."
rm -rf "$EXTERNAL_ENV"
mkdir -p "$EXTERNAL_ENV/deep-source"
echo "Deep Source File" > "$EXTERNAL_ENV/deep-source/deep_file.txt"

mkdir -p "$EXTERNAL_ENV/deep-dest-parent"
echo "Deep Dest Parent" > "$EXTERNAL_ENV/deep-dest-parent/anchor.txt"

echo "Fixtures generated successfully at $FIXTURES_DIR and $EXTERNAL_ENV"
