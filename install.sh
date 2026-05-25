#!/bin/bash

set -e

OWNER="toman92"
REPO="golem-cli"
BINARY_NAME="golem"

# OS detection
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  linux*)   OS='linux';;
  darwin*)  OS='darwin';;
  msys*|cygwin*|mingw*) OS='windows';;
  *)        echo "Error: Unsupported OS: ${OS}"; exit 1;;
esac

# Architecture detection
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)   ARCH='amd64';;
  arm64|aarch64) ARCH='arm64';;
  *)        echo "Error: Unsupported architecture: ${ARCH}"; exit 1;;
esac

echo "Detecting OS: ${OS}"
echo "Detecting Architecture: ${ARCH}"

# Fetch latest release version from GitHub API
echo "Fetching latest release version..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${OWNER}/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "${LATEST_RELEASE}" ]; then
  echo "Error: Could not find latest release for ${OWNER}/${REPO}."
  exit 1
fi

echo "Latest release: ${LATEST_RELEASE}"

FILE_EXT=""
if [ "${OS}" = "windows" ]; then
  FILE_EXT=".exe"
fi

DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST_RELEASE}/${BINARY_NAME}-${OS}-${ARCH}${FILE_EXT}"

echo "Downloading ${BINARY_NAME} from ${DOWNLOAD_URL}..."
curl -L "${DOWNLOAD_URL}" -o "${BINARY_NAME}${FILE_EXT}"

chmod +x "${BINARY_NAME}${FILE_EXT}"

# Move to /usr/local/bin if possible
INSTALL_DIR="/usr/local/bin"

if [ -w "${INSTALL_DIR}" ]; then
  mv "${BINARY_NAME}${FILE_EXT}" "${INSTALL_DIR}/${BINARY_NAME}${FILE_EXT}"
  echo "Successfully installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}${FILE_EXT}"
else
  echo "Warning: ${INSTALL_DIR} is not writable. Attempting with sudo..."
  if command -v sudo >/dev/null 2>&1; then
    sudo mv "${BINARY_NAME}${FILE_EXT}" "${INSTALL_DIR}/${BINARY_NAME}${FILE_EXT}"
    echo "Successfully installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}${FILE_EXT}"
  else
    echo "Error: Could not install to ${INSTALL_DIR}. Binary is available in current directory: $(pwd)/${BINARY_NAME}${FILE_EXT}"
    exit 1
  fi
fi

echo ""
echo "Golem installed successfully! Type 'golem' to get started."
