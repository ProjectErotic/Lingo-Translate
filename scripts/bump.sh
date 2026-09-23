#!/usr/bin/env bash
set -e

NEW_VER="$1"

if [ -z "$NEW_VER" ]; then
  echo "❌ Error: Version argument is required."
  echo "Usage: ./scripts/bump.sh <new_version> (e.g. ./scripts/bump.sh 2.3.0)"
  exit 1
fi

# Strip optional leading 'v'
NEW_VER="${NEW_VER#v}"

echo "🚀 Bumping Lingo version to ${NEW_VER}..."

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 1. Update VERSION file (Single Source of Truth)
echo "$NEW_VER" > "$ROOT_DIR/VERSION"
echo "  ✓ Updated VERSION -> ${NEW_VER}"

# 2. Update Go version fallback in pkg/version/version.go
sed -i -E "s/var Version = \".*\"/var Version = \"${NEW_VER}\"/" "$ROOT_DIR/pkg/version/version.go"
echo "  ✓ Updated pkg/version/version.go -> ${NEW_VER}"

# 3. Update frontend/package.json
if [ -f "$ROOT_DIR/frontend/package.json" ]; then
  node -e "
    const fs = require('fs');
    const p = '$ROOT_DIR/frontend/package.json';
    const pkg = JSON.parse(fs.readFileSync(p, 'utf-8'));
    pkg.version = '$NEW_VER';
    fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
  "
  echo "  ✓ Updated frontend/package.json -> ${NEW_VER}"
fi

# 4. Rebuild Frontend (picks up VERSION directly via vite.config.ts)
echo "📦 Building frontend assets..."
(cd "$ROOT_DIR/frontend" && npm run build)

# 5. Run tests to verify integrity
echo "🧪 Running Go test suite..."
(cd "$ROOT_DIR" && CGO_ENABLED=0 go test ./pkg/...)

echo ""
echo "✅ All components successfully updated to version ${NEW_VER}!"
echo "👉 To release, run:"
echo "   git add ."
echo "   git commit -m \"bump: version ${NEW_VER}\""
echo "   git push origin main"
