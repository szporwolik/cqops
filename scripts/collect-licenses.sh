#!/bin/bash
# Collect license files from all Go dependencies
set -e
cd /home/szymon/Hack/Code/cqops_app
mkdir -p licenses
rm -f licenses/-LICENSE*

MODCACHE=$(go env GOMODCACHE)

copy_license() {
  local dir="$1"
  local name="$2"
  for f in LICENSE LICENSE.md LICENSE.txt COPYING; do
    if [ -f "$dir/$f" ]; then
      cp "$dir/$f" "licenses/${name}-${f}"
      echo "  OK: ${name}"
      return 0
    fi
  done
  echo "  MISSING: ${name} (no license file found in $dir)"
  return 1
}

echo "Collecting licenses..."

copy_license "$MODCACHE/charm.land/bubbles/v2@v2.1.0" "BUBBLES-MIT"
copy_license "$MODCACHE/charm.land/bubbletea/v2@v2.0.7" "BUBBLETEA-MIT"
copy_license "$MODCACHE/charm.land/lipgloss/v2@v2.0.4" "LIPGLOSS-MIT"

NTDIR=$(find "$MODCACHE" -maxdepth 4 -path "*/NimbleMarkets/ntcharts/v2*" -type d 2>/dev/null | head -1)
[ -n "$NTDIR" ] && copy_license "$NTDIR" "NTCHARTS-MIT"

ADIFDIR=$(find "$MODCACHE" -maxdepth 4 -path "*/farmergreg/adif/v5*" -type d 2>/dev/null | head -1)
[ -n "$ADIFDIR" ] && copy_license "$ADIFDIR" "ADIF-MIT"

SPECDIR=$(find "$MODCACHE" -maxdepth 4 -path "*/farmergreg/spec/v6*" -type d 2>/dev/null | head -1)
[ -n "$SPECDIR" ] && copy_license "$SPECDIR" "SPEC-BSD3"

copy_license "$MODCACHE/golang.org/x/term@v0.44.0" "XTERM-BSD3"

copy_license "$MODCACHE/github.com/spf13/cobra@v1.10.2" "COBRA-APACHE2"
copy_license "$MODCACHE/modernc.org/sqlite@v1.52.0" "SQLITE-BSD3"
copy_license "$MODCACHE/gopkg.in/yaml.v3@v3.0.1" "YAML-MIT"

WSJTDIR=$(find "$MODCACHE" -maxdepth 4 -path "*/k0swe/wsjtx-go/v4*" -type d 2>/dev/null | head -1)
[ -n "$WSJTDIR" ] && copy_license "$WSJTDIR" "WSJTXGO-MIT"

copy_license "$MODCACHE/github.com/ftl/hamradio@v0.3.0" "HAMRADIO-MIT"
copy_license "$MODCACHE/github.com/gen2brain/beeep@v0.11.2" "BEEEP-MIT"

echo ""
echo "License files collected in licenses/:"
ls -la licenses/
echo "---DONE---"
