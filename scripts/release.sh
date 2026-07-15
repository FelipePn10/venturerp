#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
TAG="v${VERSION}"
DRY_RUN="${RELEASE_DRY_RUN:-0}"

fail() { printf 'release: %s\n' "$*" >&2; exit 1; }

[[ "${VERSION}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$ ]] ||
  fail "VERSION deve ser SemVer sem prefixo v (ex.: 1.4.0)"

[[ "$(git branch --show-current)" == "main" ]] || fail "release só pode ser criado na branch main"
[[ -z "$(git status --porcelain)" ]] || fail "worktree deve estar limpo"
git diff --quiet --exit-code
git diff --cached --quiet --exit-code
git rev-parse --verify "refs/tags/${TAG}" >/dev/null 2>&1 && fail "tag ${TAG} já existe localmente"

git fetch --tags origin
git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1 &&
  fail "tag ${TAG} já existe no origin"

printf 'release: validando %s\n' "${TAG}"
GOCACHE="${GOCACHE:-/tmp/venturerp-release-go-cache}" go test ./...
docker build \
  --build-arg "VERSION=${VERSION}" \
  --build-arg "MIN_CLIENT_VERSION=${MIN_CLIENT_VERSION:-${VERSION}}" \
  --tag "venturerp-release-check:${VERSION}" .

if [[ "${DRY_RUN}" == "1" ]]; then
  printf 'release: validação concluída; dry-run não alterou CHANGELOG, commit ou tag\n'
  exit 0
fi

PREVIOUS_TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"
RANGE="HEAD"
[[ -n "${PREVIOUS_TAG}" ]] && RANGE="${PREVIOUS_TAG}..HEAD"
NOTES="$(git log "${RANGE}" --no-merges --pretty='- %s (`%h`)' -- . ':!CHANGELOG.md')"
[[ -n "${NOTES}" ]] || NOTES="- Release ${TAG}."
DATE="$(date -u +%Y-%m-%d)"
TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT

awk -v tag="${TAG}" -v date="${DATE}" -v notes="${NOTES}" '
  /^## Unreleased$/ && !done {
    print $0 "\n\n## [" tag "] — " date "\n\n" notes
    done=1
    next
  }
  { print }
' CHANGELOG.md >"${TMP}"
mv "${TMP}" CHANGELOG.md
trap - EXIT

git add CHANGELOG.md
git commit -m "chore(release): ${TAG}"
git tag -a "${TAG}" -m "VentureERP ${TAG}"
git push --atomic origin main "${TAG}"
printf 'release: %s publicado; pipelines acionados pela tag\n' "${TAG}"
