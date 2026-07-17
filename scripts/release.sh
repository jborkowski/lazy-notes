#!/usr/bin/env bash
# Codified GitHub release flow (bump / tag / build / upload / pin).
#
# Usage:
#   ./scripts/release.sh 0.1.3
#   ./scripts/release.sh 0.1.3 --archs arm64,amd64,universal
#   ./scripts/release.sh 0.1.3 --notes "watchers"
#   ./scripts/release.sh 0.1.3 --dry-run
#
# Steps:
#   1. Preconditions (main, clean tree, tools)
#   2. Bump Formula version + tag (drop revision)
#   3. Commit + annotated tag + push
#   4. Build macOS tarballs + checksums
#   5. gh release create
#   6. Pin Formula revision to tagged SHA + push
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FORMULA_RB="Formula/lazy-notes.rb"
DIST_DIR="${DIST_DIR:-$ROOT/dist}"
ARCHS="arm64"
DRY_RUN=0
SKIP_PUSH=0
NOTES_EXTRA=""

die() { echo "error: $*" >&2; exit 1; }
log() { echo "==> $*"; }

usage() {
  cat <<'EOF'
Usage: ./scripts/release.sh X.Y.Z [--archs arm64[,amd64,universal]] [--notes TEXT] [--dry-run] [--skip-push]

Bump Formula, tag, build macOS tarballs, create GitHub release, pin Formula revision.
EOF
  exit 2
}

VERSION=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage ;;
    --archs)
      ARCHS="${2:-}"
      [[ -n "$ARCHS" ]] || die "--archs requires a value (e.g. arm64,amd64,universal)"
      shift 2
      ;;
    --skip-push) SKIP_PUSH=1; shift ;;
    --dry-run) DRY_RUN=1; shift ;;
    --notes)
      NOTES_EXTRA="${2:-}"
      shift 2
      ;;
    -*)
      die "unknown flag: $1"
      ;;
    *)
      if [[ -z "$VERSION" ]]; then
        VERSION="$1"
        shift
      else
        die "unexpected argument: $1"
      fi
      ;;
  esac
done

[[ -n "$VERSION" ]] || die "VERSION required (e.g. 0.1.3). See --help"
[[ "$VERSION" == v* ]] && VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "VERSION must look like X.Y.Z (got: $VERSION)"
TAG="v${VERSION}"

if [[ "$DRY_RUN" -eq 1 ]]; then
  log "release $TAG (archs=$ARCHS dry_run=1)"
  log "would: Formula bump → commit/tag/push → build ($ARCHS) → gh release → pin revision"
  exit 0
fi

command -v git >/dev/null || die "git required"
command -v go >/dev/null || die "go required"
command -v gh >/dev/null || die "gh required"
command -v shasum >/dev/null || die "shasum required"
command -v python3 >/dev/null || die "python3 required"

BRANCH="$(git branch --show-current)"
[[ "$BRANCH" == "main" ]] || die "must be on main (currently: $BRANCH)"
git rev-parse --abbrev-ref --symbolic-full-name @{u} >/dev/null 2>&1 || die "main has no upstream"
[[ -z "$(git status --porcelain)" ]] || die "working tree not clean; commit or stash first"
if [[ "$SKIP_PUSH" -eq 0 ]]; then
  git fetch origin main --tags
  LOCAL="$(git rev-parse HEAD)"
  REMOTE="$(git rev-parse origin/main)"
  [[ "$LOCAL" == "$REMOTE" ]] || die "main is not in sync with origin/main (pull/push first)"
fi
if git rev-parse "$TAG" >/dev/null 2>&1; then
  die "tag $TAG already exists locally"
fi
if [[ "$SKIP_PUSH" -eq 0 ]] && git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
  die "tag $TAG already exists on origin"
fi
if [[ "$SKIP_PUSH" -eq 0 ]] && gh release view "$TAG" >/dev/null 2>&1; then
  die "GitHub release $TAG already exists"
fi

bump_formula() {
  local mode="$1" # bump | pin
  local sha="${2:-}"
  python3 - "$FORMULA_RB" "$VERSION" "$TAG" "$mode" "$sha" <<'PY'
import pathlib, re, sys

path, version, tag, mode, sha = sys.argv[1:6]
text = pathlib.Path(path).read_text()
text, n = re.subn(r'(version\s+")[^"]+(")', rf"\g<1>{version}\2", text, count=1)
if n != 1:
    raise SystemExit("could not update version in Formula")

pat = re.compile(
    r'url "https://github\.com/jborkowski/lazy-notes\.git",\n'
    r'(?:[ \t]*tag:\s+"[^"]+",?\n)?'
    r'(?:[ \t]*revision:\s+"[0-9a-f]+"\n)?'
)
if mode == "bump":
    repl = (
        'url "https://github.com/jborkowski/lazy-notes.git",\n'
        f'        tag: "{tag}"\n'
    )
elif mode == "pin":
    if not re.fullmatch(r"[0-9a-f]{7,40}", sha):
        raise SystemExit(f"pin requires git sha, got {sha!r}")
    repl = (
        'url "https://github.com/jborkowski/lazy-notes.git",\n'
        f'        tag:      "{tag}",\n'
        f'        revision: "{sha}"\n'
    )
else:
    raise SystemExit(f"unknown mode {mode}")

text, n = pat.subn(repl, text, count=1)
if n != 1:
    raise SystemExit("could not update git tag/revision block in Formula")
pathlib.Path(path).write_text(text)
print(f"updated {path} ({mode})")
PY
}

build_one() {
  local goarch="$1"
  local name arch_label outdir tarball
  case "$goarch" in
    arm64) arch_label="darwin-arm64" ;;
    amd64) arch_label="darwin-amd64" ;;
    universal) arch_label="darwin-universal" ;;
    *) die "unsupported arch: $goarch (use arm64, amd64, universal)" ;;
  esac
  name="lazy-notes-${TAG}-${arch_label}"
  outdir="$DIST_DIR/$name"
  tarball="$DIST_DIR/${name}.tar.gz"
  rm -rf "$outdir"
  mkdir -p "$outdir/config"
  if [[ "$goarch" == "universal" ]]; then
    local tmp="$DIST_DIR/.universal-build"
    rm -rf "$tmp"
    mkdir -p "$tmp"
    GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "$tmp/lazy-notes-arm64" ./cmd/lazy-notes
    GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "$tmp/lazy-notes-amd64" ./cmd/lazy-notes
    lipo -create -output "$outdir/lazy-notes" "$tmp/lazy-notes-arm64" "$tmp/lazy-notes-amd64"
    rm -rf "$tmp"
  else
    GOOS=darwin GOARCH="$goarch" go build -ldflags="-s -w" -o "$outdir/lazy-notes" ./cmd/lazy-notes
  fi
  /bin/cp README.md "$outdir/"
  rsync -a --delete config/ "$outdir/config/"
  tar -C "$DIST_DIR" -czf "$tarball" "$name"
  echo "$tarball"
}

log "release $TAG (archs=$ARCHS)"

log "bump Formula version → $VERSION"
bump_formula bump
git add "$FORMULA_RB"
git commit -m "$(cat <<EOF
release: bump to ${TAG}

EOF
)"
RELEASE_SHA="$(git rev-parse HEAD)"
git tag -a "$TAG" -m "release: ${TAG}"
log "tagged $TAG @ $RELEASE_SHA"

if [[ "$SKIP_PUSH" -eq 0 ]]; then
  log "push main + $TAG"
  git push origin main
  git push origin "$TAG"
fi

log "build release archives → $DIST_DIR"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
ASSETS=()
IFS=',' read -r -a arch_list <<<"$ARCHS"
for a in "${arch_list[@]}"; do
  a="$(echo "$a" | tr -d '[:space:]')"
  [[ -n "$a" ]] || continue
  log "building $a"
  tarball="$(build_one "$a")"
  ASSETS+=("$tarball")
done
(
  cd "$DIST_DIR"
  : > checksums.txt
  for f in "${ASSETS[@]}"; do
    shasum -a 256 "$(basename "$f")" >> checksums.txt
  done
)
ASSETS+=("$DIST_DIR/checksums.txt")

NOTES="macOS release ${TAG}."
if [[ -n "$NOTES_EXTRA" ]]; then
  NOTES="${NOTES}

## Changes

${NOTES_EXTRA}"
fi

log "create GitHub release $TAG"
gh release create "$TAG" \
  --title "$TAG" \
  --notes "$NOTES" \
  "${ASSETS[@]}"

log "pin Formula revision → $RELEASE_SHA"
bump_formula pin "$RELEASE_SHA"
git add "$FORMULA_RB"
git commit -m "$(cat <<EOF
chore: pin Formula git revision to ${TAG}

EOF
)"
if [[ "$SKIP_PUSH" -eq 0 ]]; then
  git push origin main
fi

log "done: ${TAG}"
echo "release: $(gh release view "$TAG" --json url -q .url)"
echo "sha:     $RELEASE_SHA"
echo "assets:  ${ASSETS[*]}"
