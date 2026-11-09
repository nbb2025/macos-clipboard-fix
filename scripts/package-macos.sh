#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."
if [[ "$(uname -s)" != Darwin ]]; then
  echo 'Packaging requires macOS.' >&2
  exit 1
fi

version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' build/darwin/Info.plist)
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'Invalid application version.' >&2
  exit 1
fi
if [[ "${GITHUB_REF_TYPE:-}" == tag && "${GITHUB_REF_NAME:-}" != "v$version" ]]; then
  echo 'Release tag must match the application version.' >&2
  exit 1
fi

app='bin/Clipboard Fix.app'
executable="$app/Contents/MacOS/macos-clipboard-fix"
test -f "$executable"
codesign --verify --deep --strict "$app"
architectures=$(lipo -archs "$executable")
for architecture in arm64 x86_64; do
  if [[ " $architectures " != *" $architecture "* ]]; then
    echo "Missing architecture: $architecture" >&2
    exit 1
  fi
done

out="${1:-dist}"
mkdir -p "$out"
out=$(cd "$out" && pwd)
base="Clipboard-Fix-$version-universal"
stage=$(mktemp -d "${TMPDIR:-/tmp}/clipboard-fix-package.XXXXXX")
trap 'rm -rf -- "$stage"' EXIT
mkdir "$stage/volume" "$stage/verify"
ditto "$app" "$stage/volume/Clipboard Fix.app"
ln -s /Applications "$stage/volume/Applications"
codesign --verify --deep --strict "$stage/volume/Clipboard Fix.app"

ditto -c -k --sequesterRsrc --keepParent "$app" "$out/$base.zip"
ditto -x -k "$out/$base.zip" "$stage/verify"
codesign --verify --deep --strict "$stage/verify/Clipboard Fix.app"
hdiutil create -volname 'Clipboard Fix' -srcfolder "$stage/volume" -format UDZO -ov "$out/$base.dmg"
hdiutil verify "$out/$base.dmg"
(
  cd "$out"
  shasum -a 256 "$base.zip" "$base.dmg" > SHA256SUMS
)
echo "Packages verified: $out"
