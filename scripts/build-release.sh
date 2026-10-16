#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
version="v0.1.0"
if [ "$#" -gt 0 ]; then version="$1"; fi

mkdir -p dist
release_dir="$(pwd)/dist"
: > "$release_dir/SHA256SUMS"

for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64 windows_arm64; do
  os="$(printf '%s' "$target" | cut -d_ -f1)"
  arch="$(printf '%s' "$target" | cut -d_ -f2)"
  printf -v base 'pastecrate_%s_%s_%s' "$version" "$os" "$arch"
  staged="$(mktemp -d)"
  binary="$staged/pastecrate"
  if [ "$os" = windows ]; then binary="$staged/pastecrate.exe"; fi

  echo "Building $os/$arch..."
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$binary" .
  cp README.md LICENSE "$staged/"

  if [ "$os" = windows ]; then
    (cd "$staged" && zip -q "$release_dir/$base.zip" pastecrate.exe README.md LICENSE)
    archive="$base.zip"
  else
    archive="$base.tar.gz"
    tar -C "$staged" -czf "$release_dir/$archive" pastecrate README.md LICENSE
  fi

  rm -rf "$staged"
  (cd "$release_dir" && sha256sum "$archive" >> SHA256SUMS)
done

echo "Release archives and SHA256SUMS are in $release_dir"
