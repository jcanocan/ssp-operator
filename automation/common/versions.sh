#!/bin/bash

set -e
set -o pipefail

# This function will be used in release branches
function latest_patch_version() {
  local repo="$1"
  local minor_version="$2"

  # The loop is necessary, because GitHub API call cannot return more than 100 items
  local latest_version=""
  local page=1
  while true ; do
    # Declared separately to not mask return value
    local versions_in_page
    versions_in_page=$(
      curl --fail -s "https://api.github.com/repos/kubevirt/${repo}/releases?per_page=100&page=${page}" |
      jq '.[] | select(.prerelease==false) | .tag_name' |
      tr -d '"'
    )
    if [ $? -ne 0 ]; then
      return 1
    fi

    if [ -z "${versions_in_page}" ]; then
      break
    fi

    latest_version=$(
      echo "${versions_in_page} ${latest_version}" |
      tr " " "\n" |
      grep "^${minor_version}\\." |
      sort --version-sort |
      tail -n1
    )

    ((++page))
  done

  echo "${latest_version}"
}

function latest_version() {
  local org="$1"
  local repo="$2"

  # The API call sorts releases by creation timestamp, so it is enough to request only a few latest ones.
  curl --fail -s "https://api.github.com/repos/${org}/${repo}/releases" | \
    jq '.[] | select(.prerelease==false) | .tag_name' | \
    tr -d '"' | \
    sort --version-sort | \
    tail -n1
}

# Latest released Kubevirt version
# KUBEVIRT_VERSION=$(latest_version "kubevirt" "kubevirt")
# FIXME(lyarwood) Replace with above once v1.0.0 is released
KUBEVIRT_VERSION=v1.0.0-rc.0

# Latest released CDI version
CDI_VERSION=$(latest_version "kubevirt" "containerized-data-importer")

# Latest released Tekton version
TEKTON_VERSION=$(latest_version "tektoncd" "operator")
