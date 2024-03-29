BEAT_NAME?=assetbeat
BEAT_TITLE?=Inputruner
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"
ES_BEATS?=..
EXCLUDE_COMMON_UPDATE_TARGET=true

include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update

# Creates a new module. Requires the params MODULE.
.PHONY: create-module
create-module: mage
	mage generate:module

# Creates a new fileset. Requires the params MODULE and FILESET.
.PHONY: create-fileset
create-fileset: mage
	mage generate:fileset

# Creates a fields.yml based on a pipeline.json file. Requires the params MODULE and FILESET.
.PHONY: create-fields
create-fields: mage
	mage generate:fields
