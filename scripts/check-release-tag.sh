#!/usr/bin/env bash
# check-release-tag.sh VERSION COMMIT
#
# Fails when the tag v<VERSION> already exists and does not point at COMMIT.
#
# Why this exists: the GitHub Releases API creates a missing tag at the release
# request's target_commitish, but IGNORES target_commitish when the tag already
# exists. A release published under such a tag therefore carries artifacts built
# from COMMIT while the tag — and the source archives GitHub serves for it —
# point at some other commit. A manual run from a branch (the only way this
# workflow starts) is exactly the case where that happens silently.
#
# Exits 0 when the tag is missing (it will be created at COMMIT) or when it
# already points at COMMIT.
set -euo pipefail

V="${1:?version required}"
SHA="${2:?commit required}"
TAG="v${V}"

if [ -z "$(git rev-parse -q --verify "refs/tags/${TAG}" 2>/dev/null || true)" ]; then
	echo "tag ${TAG} does not exist yet — it will be created at ${SHA}"
	exit 0
fi

# Peels an annotated tag to its commit; a lightweight tag resolves directly.
TAG_SHA="$(git rev-parse -q --verify "refs/tags/${TAG}^{commit}" 2>/dev/null || true)"
if [ -z "${TAG_SHA}" ]; then
	echo "::error::tag ${TAG} exists but does not resolve to a commit"
	exit 1
fi

if [ "${TAG_SHA}" != "${SHA}" ]; then
	{
		echo "::error::tag ${TAG} already exists at ${TAG_SHA}, but this run built ${SHA}."
		echo "The GitHub Releases API ignores target_commitish for an existing tag, so these artifacts"
		echo "would be published against a commit that is not the tagged one (nor its source archives)."
		echo "Re-run the workflow from ${TAG_SHA}, or move the tag to the built commit:"
		echo "  git tag -f ${TAG} ${SHA} && git push --force origin ${TAG}"
	} >&2
	exit 1
fi

echo "tag ${TAG} already exists at ${SHA} — matches the built commit"
