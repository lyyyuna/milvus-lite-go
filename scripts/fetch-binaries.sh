#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-2.5.1}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLATFORM_DIR="${SCRIPT_DIR}/../platform"

PLATFORMS="darwin-arm64 darwin-amd64 linux-amd64 linux-arm64"
wheel_platform_for() {
  case "$1" in
    darwin-arm64) echo "macosx_11_0_arm64" ;;
    darwin-amd64) echo "macosx_10_9_x86_64" ;;
    linux-amd64)  echo "manylinux2014_x86_64" ;;
    linux-arm64)  echo "manylinux2014_aarch64" ;;
  esac
}

TMP_DIR=$(mktemp -d)
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

for platform in $PLATFORMS; do
  wp=$(wheel_platform_for "$platform")
  wheel_name="milvus_lite-${VERSION}-py3-none-${wp}.whl"
  lib_dir="${PLATFORM_DIR}/${platform}/lib"

  echo "=== ${platform} ==="

  rm -rf "$lib_dir"
  mkdir -p "$lib_dir"

  echo "  Downloading ${wheel_name}..."
  pip3 download "milvus-lite==${VERSION}" \
    --no-deps --only-binary=:all: \
    --platform "$wp" \
    --python-version 3.11 \
    -d "$TMP_DIR" \
    --no-cache-dir \
    -q

  echo "  Extracting lib..."
  unzip -o -q "${TMP_DIR}/${wheel_name}" "milvus_lite/lib/*" -d "${TMP_DIR}/extract_${platform}"
  cp -R "${TMP_DIR}/extract_${platform}/milvus_lite/lib/"* "$lib_dir/"
  chmod +x "${lib_dir}/milvus"

  # Remove placeholder if exists
  rm -f "${lib_dir}/PLACEHOLDER"

  echo "  Done: $(ls "$lib_dir" | wc -l | tr -d ' ') files"
  echo ""
done

echo "All platforms fetched successfully."
