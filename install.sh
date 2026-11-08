#!/usr/bin/env bash
# Install the newest published CLI, its managed tools, and verify the result.
set -euo pipefail

case "${1:-}" in
  '') ;;
  -h|--help)
    printf '%s\n' 'Usage: bash install.sh' \
      'Installs the newest published release (including prereleases), then runs tools install and doctor.' \
      "Destination: \$XDG_BIN_HOME (default: \$HOME/.local/bin)." \
      'Requires Linux, curl, jq, tar, sha256sum and install; no sudo or Go.'
    exit 0 ;;
  *) printf 'Unexpected argument: %s\n' "$1" >&2; exit 2 ;;
esac
[[ $# -le 1 ]] || { printf 'Too many arguments\n' >&2; exit 2; }
[[ $(uname -s) == Linux ]] || { printf 'Only Linux is supported\n' >&2; exit 1; }
case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) printf 'Unsupported architecture\n' >&2; exit 1 ;;
esac
destination=${XDG_BIN_HOME:-${HOME:?Set HOME or XDG_BIN_HOME}/.local/bin}
for dependency in curl jq tar sha256sum install mktemp; do
  command -v "$dependency" >/dev/null || { printf 'Missing dependency: %s\n' "$dependency" >&2; exit 1; }
done
directory=$(mktemp -d)
staged=
trap 'rm -rf -- "$directory"; if [[ -n "$staged" ]]; then rm -f -- "$staged"; fi' EXIT
fetch() {
  curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    --connect-timeout 20 --max-time 300 --retry 2 "$@"
}
# /releases/latest excludes prereleases. Inspect all pages and use publication
# time, not tag order or the commit creation date.
page=1
while :; do
  fetch -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/ever-guild/quality-bash/releases?per_page=100&page=$page" \
    -o "$directory/releases-$page.json"
  count=$(jq -er 'if type == "array" then length else error("Invalid releases response") end' "$directory/releases-$page.json")
  [[ $count -eq 100 ]] || break
  page=$((page + 1))
done
jq -es 'add | map(select(.draft == false and (.published_at | type == "string"))) |
  if length == 0 then error("No published releases") else max_by([.published_at, .id]) end' \
  "$directory"/releases-*.json > "$directory/release.json"
version=$(jq -er '.tag_name | select(type == "string")' "$directory/release.json")
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || { printf 'Invalid release tag\n' >&2; exit 1; }
asset="quality-bash-$version-linux-$arch.tar.gz"
digest=$(jq -er --arg asset "$asset" '[.assets[] | select(.name == $asset and .state == "uploaded")] |
  if length == 1 then .[0].digest else error("Release must have exactly one matching CLI archive") end' "$directory/release.json")
[[ $digest =~ ^sha256:[0-9a-f]{64}$ ]] || { printf 'Release archive has no valid SHA-256 digest\n' >&2; exit 1; }
fetch "https://github.com/ever-guild/quality-bash/releases/download/$version/$asset" -o "$directory/$asset"
(
  cd "$directory"
  printf '%s  %s\n' "${digest#sha256:}" "$asset" > selected.sha256
  sha256sum --check selected.sha256
  tar -xzf "$asset" quality-bash
)
[[ -f "$directory/quality-bash" && ! -L "$directory/quality-bash" ]] || { printf 'Archive must contain a regular CLI file\n' >&2; exit 1; }
mkdir -p -- "$destination"
staged=$(mktemp "$destination/.quality-bash.XXXXXXXX")
install -m 0755 -- "$directory/quality-bash" "$staged"
mv -fT -- "$staged" "$destination/quality-bash"
staged=
printf 'Installed %s to %s/quality-bash\n' "$version" "$destination"
"$destination/quality-bash" tools install
"$destination/quality-bash" doctor
printf 'Installation verified. Add %s to PATH.\n' "$destination"
