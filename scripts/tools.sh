#!/bin/bash

set -e

source scripts/func.sh

# Note: the set [-/+] x is purely there to turn on and off outputting of the commands being executed.
if [ "${DEBUG}" = "true" ]; then
  set -x
fi

mkdir -p tmp var
touch .env.local

case "$1" in

"prep")
  export VERSION=latest
  scripts/tools.sh install
  export GOPROXY=direct
  go install github.com/codemityio/auxilium@latest
  go install github.com/codemityio/notatio@latest
  go install github.com/codemityio/goforma@latest
  ;;

"cmd")
  docker run --rm \
    --name "${BASE_NAME}-cmd" \
    -e DEBUG \
    -e GOBIN="${PWD}/bin" \
    -v "${PWD}:${PWD}" \
    -w "${PWD}" \
    codemityio/golang-dev:latest sh -c "${COMMAND}"
  ;;

"run")
  case "$2" in

  "go")
    set -a
    source .env
    source .env.local
    set +a
    go run . ${FLAGS} ${COMMAND}
    ;;

  "container")
    docker run --rm --env-file=.env --env-file=.env.local "${IMAGE_NAME}:latest" ${FLAGS} ${COMMAND}
    ;;

  *)
    echo "error: incorrect '$2' subcommand..."
    ;;

  esac
  ;;

"exec")
  set -a
  source .env
  source .env.local
  set +a
  bin/app ${FLAGS} ${COMMAND}
  ;;

"cleanup")
  find . -iname '*_mock_test.go' -exec rm {} \;
  find . -iname '*.so' -exec rm {} \;
  git clean -dXf pkg internal cmd
  rm -Rf tmp var bin
  ;;

"update")
  COMMAND="go mod init github.com/${VENDOR}/$(basename "${PWD}")"
  export COMMAND
  rm -rf go.* && scripts/tools.sh cmd
  scripts/tools.sh fmt
  ;;

"gen")
  go generate -v ./...
  cat >"var.go" <<EOF
//nolint:gochecknoglobals
package main

var (
	name        = "${BASE_NAME}"
	version     = ""
	copyright   = "${VENDOR}"
	authorName  = "${VENDOR}"
	authorEmail = ""
	buildTime   = ""
)
EOF
  ;;

"fmt")
  go mod tidy
  goimports -w .
  gofumpt -l -w .
  go vet ./...
  ;;

"statan")
  statan "./..." ""
  ;;

"statan-fix")
  statan "./..." "--fix"
  ;;

"test")
  go test -failfast -v -covermode=count -coverprofile=tmp/coverage.out ./...
  ;;

"test-race")
  go test -race -failfast -v -covermode=atomic -coverprofile=tmp/coverage.out ./...
  ;;

"cov")
  go tool cover -func="tmp/coverage.out" -o tmp/coverage.in
  goforma badge \
    --document-path=README.md \
    --id=coverage-badge-do-not-edit \
    coverage \
    --cov-file-path=tmp/coverage.in \
    --minimum="${MINIMUM_COVERAGE}"
  ;;

"cov-report")
  go tool cover -func=tmp/coverage.out
  ;;

"cov-open")
  go tool cover -html=tmp/coverage.out -o tmp/coverage.html
  open tmp/coverage.html
  ;;

"diff")
  (git diff --quiet && git diff --cached --quiet && [ -z "$(git ls-files --others --exclude-standard)" ]) || {
    echo "error: changes detected..."
    exit 1
  }
  ;;

"version")
  tag=$(git tag -l | sort -V | tail -n1)
  echo "${tag:-latest}"
  ;;

"next")
  # get latest v-tag (supports vX.Y.Z and vX.Y.Z-preN) and bump prerelease or patch
  latest=$(git tag -l "v*" | sort -V | tail -n1)

  if [[ $latest =~ ^v([0-9]+\.[0-9]+\.[0-9]+)-([a-zA-Z]+)([0-9]+)$ ]]; then
    next="v${BASH_REMATCH[1]}-${BASH_REMATCH[2]}$((BASH_REMATCH[3] + 1))"
  elif [[ $latest =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    next="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))"
  else
    echo "error: unsupported tag format: $latest"
    exit 1
  fi

  git tag "$next"
  echo "Tagged: $next"
  ;;

"go")
  go generate -skip=mockgen -v ./...
  go build -ldflags "\
-X 'main.name=${BASE_NAME}' \
-X 'main.version=${VERSION}' \
-X 'main.copyright=${VENDOR}' \
-X 'main.authorName=${VENDOR}' \
-X 'main.buildTime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'\
" -o bin/app .
  ;;

"install")
  go install -ldflags "\
-X 'main.name=${BASE_NAME}' \
-X 'main.version=${VERSION}' \
-X 'main.copyright=${VENDOR}' \
-X 'main.authorName=${VENDOR}' \
-X 'main.buildTime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'\
" .
  ;;

*)
  echo "error: incorrect '$1' command..."
  ;;

esac
