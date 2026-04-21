#!/bin/bash

entries=$(grep -E '^[a-zA-Z0-9_-]+:.*?## .*$' Makefile | sort)

if [ ! -t 0 ] || [ "${CI}" = "true" ]; then
  echo "${entries}" | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $1, $2}'
  exit 0
fi

if ! command -v auxilium &>/dev/null; then
  echo "Please install (https://github.com/codemityio/auxilium) tools..."
else
  list=$(echo "${entries}" | awk 'BEGIN {FS = ":.*?## "}; {printf "%s %s\n", $1, $2}')
  targets=$(auxilium select --select-name-label="Target" --select-value-label="Description" --list="$list" ${SIZE:+--size=${SIZE}})
  read -r -a parts <<<"$targets"
  if [[ ${parts[0]} == "" ]]; then exit 0; fi
  # shellcheck disable=SC2128
  $1 "${parts}"
fi
