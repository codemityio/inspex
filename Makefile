.PHONY: build cmd docs tools

-include Makefile.env
-include Makefile.env.local

%: # Silence errors about non existing targets
	@true

default: # A default target to initiate interactive menu
	@source scripts/func.sh && check "git docker go goimports gofumpt"
	@scripts/default.sh make

help: ## Prints help for targets with comments
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

version: ## Print the most recent version
	@scripts/tools.sh version

next: ## Create a new version (bump prerelease or patch)
	@scripts/tools.sh next

check: prep gen fmt statan test cov diff cleanup ## Run all CI required targets

###########
## D E V ##
###########

prep: ## Prepare dev tools
	@scripts/tools.sh prep

cmd: ## Run a command passed as COMMAND= value (e.g. make cmd COMMAND="make check")
	@scripts/tools.sh cmd

run-go: ## Run go (use FLAGS= and COMMAND= environment variables to pass main command flags and subcommand with flags when needed)
	@scripts/tools.sh run go

run-container: ## Run container (use FLAGS= and COMMAND= environment variables to pass main command flags and subcommand with flags when needed)
	@scripts/tools.sh run container

exec: ## Execute built bin (use FLAGS= and COMMAND= environment variables to pass main command flags and subcommand with flags when needed)
	@scripts/tools.sh exec

cleanup: ## Cleanup project
	@scripts/tools.sh cleanup

update: ## Update all dependencies
	@scripts/tools.sh update

vendor: ## Run go mod vendor
	@go mod vendor -v

gen: ## Go generate
	@scripts/tools.sh gen

fmt: ## Format code
	@scripts/tools.sh fmt

statan: ## Analyse code
	@scripts/tools.sh statan

statan-fix: ## Analyse code and fix
	@scripts/tools.sh statan-fix

test: ## Run tests
	@scripts/tools.sh test

test-race: ## Run race tests
	@scripts/tools.sh test-race

cov: ## Check coverage
	@scripts/tools.sh cov

cov-report: ## Check coverage report
	@scripts/tools.sh cov-report

cov-open: ## Inspect coverage in the browser
	@scripts/tools.sh cov-open

diff: ## Check diff to ensure this project consistency
	@scripts/tools.sh diff

###############
## B U I L D ##
###############

go: ## Build Go
	@scripts/tools.sh go

install: ## Install binary locally
	@scripts/tools.sh install

build: ## Build container image
	@scripts/docker.sh build

buildx: ## Build container multi platform images and push
	@scripts/docker.sh buildx

#############
## D O C S ##
#############

docs: docs-main ## Generate all docs
	@PACKAGES='$(shell find "${PWD}/pkg" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; 2>/dev/null)' make docs-uml docs-depgraph docs-pkg docs-render
	@PACKAGES='$(shell find "${PWD}/cmd" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; 2>/dev/null)' make docs-cmd

docs-uml: ## Generate UML documentation
	@scripts/docs.sh uml

docs-depgraph: ## Generate dependency graph
	@scripts/docs.sh depgraph

docs-cmd: ## Generate cmd docs
	@scripts/docs.sh cmd

docs-pkg: ## Generate pkg docs
	@scripts/docs.sh pkg

docs-render: ## Render diagrams
	@scripts/docs.sh render

docs-main: ## Generate main docs
	@scripts/docs.sh main

##########################
## D A N G E R  Z O N E ##
##########################

reset: ## Stop and remove project containers, remove project volumes, remove project images
	@docker ps -a --filter "name=${BASE_NAME}" --format "{{.ID}}" | xargs -r docker stop
	@docker ps -a --filter "name=${BASE_NAME}" --format "{{.ID}}" | xargs -r docker rm
	@docker volume ls --filter "name=${BASE_NAME}" --format "{{.Name}}" | xargs -r docker volume rm
	@docker images --format "{{.Repository}}:{{.Tag}} {{.ID}}" | grep "^${IMAGE_NAME}" | cut -d' ' -f2 | xargs -r docker rmi -f
	@docker system prune -f
