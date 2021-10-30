TAG         ?= $(shell git rev-parse HEAD)
KO_DOCKER_REPO ?= localhost:5000

SOURCES  := $(notdir $(wildcard cmd/*))
SCRIPTDIR :="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

.PHONY: publish
publish: publish-snmp publish-direktiv

.PHONY: push-%
publish-%:
	@echo "building $(TAG) with tag $*"
	@KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko publish ./cmd/$*-source/ -B -t $(TAG)
