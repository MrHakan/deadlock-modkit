#!/bin/sh
# Publishes the GitLab release for the tag being built ($CI_COMMIT_TAG), from a CI job.
#
# Every file in dist/ is uploaded to the project's generic package registry together with a
# "<file>.sha256" (sha256sum's format), and the release links them all. "dlmod update" refuses a
# HUD whose release has no .sha256 next to it. The release notes come from
# release-notes/<tag>.md: its first line ("# <title>") is the title, the rest the description.
set -eu

version=$CI_COMMIT_TAG
notes="release-notes/$version.md"
packages="$CI_API_V4_URL/projects/$CI_PROJECT_ID/packages/generic/deadlock-modkit/$version"

cd dist
for file in *; do
	case "$file" in *.sha256) continue ;; esac
	sha256sum "$file" > "$file.sha256"
done
links='[]'
for file in *; do
	curl --fail --silent --show-error --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file "$file" "$packages/$file"
	echo
	links=$(printf '%s' "$links" | jq --arg name "$file" --arg url "$packages/$file" \
		'. + [{name: $name, url: $url, direct_asset_path: ("/" + $name),
		       link_type: (if ($name | endswith(".sha256")) then "other" else "package" end)}]')
done
cd ..

title=$(head -n 1 "$notes" | sed 's/^# //')
tail -n +3 "$notes" > notes-body.md
jq -n --arg tag "$version" --arg name "$title" --rawfile description notes-body.md \
	--arg ref "$CI_COMMIT_SHA" --argjson links "$links" \
	'{tag_name: $tag, name: $name, description: $description, ref: $ref, assets: {links: $links}}' > release.json
curl --fail-with-body --silent --show-error --header "JOB-TOKEN: $CI_JOB_TOKEN" \
	--header "Content-Type: application/json" --data @release.json \
	"$CI_API_V4_URL/projects/$CI_PROJECT_ID/releases"
echo
echo "Published $version: $CI_PROJECT_URL/-/releases/$version"
