TOPTARGETS := all deps format lint test test-short clean upgrade
MODS ?= $(shell go list -f '{{.Dir}}' -m)
VERSION := $(shell cat VERSION)
MAJOR_VERSION = $(word 1, $(subst ., ,$(VERSION)))
MINOR_VERSION = $(word 2, $(subst ., ,$(VERSION)))
PATCH_VERSION = $(word 3, $(subst ., ,$(word 1,$(subst -, , $(VERSION)))))
NEXT_VERSION ?= $(MAJOR_VERSION).$(MINOR_VERSION).$(shell echo $$(( $(PATCH_VERSION) + 1)) )

default: all

$(TOPTARGETS): $(MODS)
$(MODS):
	@echo "## $$(basename $@)"
	@$(MAKE) -C $@ $(MAKECMDGOALS)
	@echo ""

.PHONY: $(TOPTARGETS) $(MODS) default

dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@v2.5.0
	go install github.com/boumenot/gocover-cobertura@v1.2.0
	go install golang.org/x/pkgsite/cmd/pkgsite@latest

deps: dev-deps
	go work sync

test:
	cat */coverage.txt | gocover-cobertura  > coverage.xml
test-short:
	cat */coverage.txt | gocover-cobertura  > coverage.xml


bump-version:
ifneq ($(shell git status -s),)
	@echo "Unable to promote a dirty workspace"
	@exit 1
endif
	@echo -n $(NEXT_VERSION) > VERSION
	git add VERSION
	@echo -n $(NEXT_VERSION) > codecatalyst-runner/VERSION
	git add codecatalyst-runner/VERSION
	git commit -a -m "chore: bump version to $(NEXT_VERSION)"
	git tag -a $(NEXT_VERSION) -m $(NEXT_VERSION)
	git push -o ci.skip origin HEAD:main
	git push origin $(NEXT_VERSION)

.PHONY: docs
docs:
	pkgsite -open .

.PHONY: attribution
attribution:
	go-licenses report ./command-runner/... ./codecatalyst-runner/...  --template attribution.tpl |grep -v codecatalyst-runner-cli > ATTRIBUTION.txt

upgrade:
	go work sync
	make attribution
