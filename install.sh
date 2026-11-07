#!/usr/bin/env bash
# Install the pinned CLI. Tool installation remains a separate explicit command.
set -euo pipefail

version=v1.0.0-alpha.2
case "${1:-}" in
  '') ;;
  -h|--help)
    printf '%s\n' 'Usage: bash install.sh' "Installs quality-bash into \$XDG_BIN_HOME (default: \$HOME/.local/bin)." 'Requires Linux, curl, tar, sha256sum and install; no sudo or Go.'
    exit 0 ;;
  *) printf 'Unexpected argument: %s\n' "$1" >&2; exit 2 ;;
esac
[[ $# -le 1 ]] || { printf 'Too many arguments\n' >&2; exit 2; }
[[ $(uname -s) == Linux ]] || { printf 'Only Linux is supported\n' >&2; exit 1; }
case "$(uname -m)" in
  x86_64) arch=amd64; digest=dd6ed1b1b2c6009ebbdd5e14b5f1c22b0e010d37cd25c194987df118f3df6c49 ;;
  aarch64|arm64) arch=arm64; digest=ff3f5c1e1feafcee3f291a8869a6ddaeb82a8bfb3694912d34b773e62f9b5242 ;;
  *) printf 'Unsupported architecture\n' >&2; exit 1 ;;
esac
destination=${XDG_BIN_HOME:-${HOME:?Set HOME or XDG_BIN_HOME}/.local/bin}
for dependency in curl tar sha256sum install mktemp; do
  command -v "$dependency" >/dev/null || { printf 'Missing dependency: %s\n' "$dependency" >&2; exit 1; }
done
directory=$(mktemp -d)
trap 'rm -rf -- "$directory"' EXIT
asset="quality-bash-$version-linux-$arch.tar.gz"
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
  "https://github.com/ever-guild/quality-bash/releases/download/$version/$asset" -o "$directory/$asset"
(
  cd "$directory"
  printf '%s  %s\n' "$digest" "$asset" > selected.sha256
  sha256sum --check selected.sha256
  tar -xzf "$asset" quality-bash
)
mkdir -p -- "$destination"
install -m 0755 -- "$directory/quality-bash" "$destination/quality-bash"
printf 'Installed %s to %s/quality-bash\n' "$version" "$destination"
printf 'Add that directory to PATH, then run: quality-bash tools install\n'
