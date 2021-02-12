#!/usr/bin/env bash
set -o errexit -o pipefail -o nounset -o xtrace

targets=(
  #  "linux/386"
  "linux/amd64"
  #  "linux/arm"
  #  "linux/arm64"
  #  "freebsd/386"
  #  "freebsd/amd64"
  #  "freebsd/arm"
  #  "freebsd/arm64"
  "darwin/amd64"
  #  "darwin/arm64"
  #  "windows/386"
  #  "windows/amd64"
)

echo "" >pidfile

for target in "${targets[@]}"; do
  os=$(cut -d '/' -f 1 <<<"${target}")
  arch=$(cut -d '/' -f 2 <<<"${target}")
  suffix=$([ "${os}" == "windows" ] && echo ".exe" || echo "")

  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
    -mod vendor \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "bin/${BIN}_${os}_${arch}${suffix}" &

  echo $! >>pidfile
done

wait $(<pidfile)
rm pidfile
