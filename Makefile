# lamp build rules.
GO ?= go
# Allow setting of go build flags from the command line.
GOFLAGS :=
# Set to 1 to use static linking for all builds (including tests).
STATIC :=
# The lamp image to be used for starting Docker containers
# during acceptance tests. Usually lampdb/lamp{,-dev}
# depending on the context.
lamp_IMAGE :=

RUN := run

# Variables to be overridden on the command line, e.g.
#   make test PKG=./storage TESTFLAGS=--vmodule=multiraft=1
PKG          := ./...
TAGS         :=
TESTS        := ".*"
TESTTIMEOUT  := 1m10s
CPUS         := 1
RACETIMEOUT  := 5m
BENCHTIMEOUT := 5m
TESTFLAGS    :=

ifeq ($(STATIC),1)
# The netgo build tag instructs the net package to try to build a
# Go-only resolver.
TAGS += netgo
# The installsuffix makes sure we actually get the netgo build, see
# https://github.com/golang/go/issues/9369#issuecomment-69864440
GOFLAGS += -installsuffix netgo
LDFLAGS += -extldflags "-static"
endif

.PHONY: all
all: build test check

# On a release build, rebuild everything (except stdlib)
# to make sure that the 'release' build tag is taken
# into account.
.PHONY: release
release: TAGS += release
release: GOFLAGS += -a
release: build

.PHONY: build
build: LDFLAGS += -X github.com/lampdb/lamp/util.buildTag "$(shell git describe --dirty)"
build: LDFLAGS += -X github.com/lampdb/lamp/util.buildTime "$(shell date -u '+%Y/%m/%d %H:%M:%S')"
build: LDFLAGS += -X github.com/lampdb/lamp/util.buildDeps "$(shell GOPATH=${GOPATH} build/depvers.sh)"
build:
	$(GO) build -tags '$(TAGS)' $(GOFLAGS) -ldflags '$(LDFLAGS)' -v -i -o lamp

.PHONY: install
install: LDFLAGS += -X github.com/lampdb/lamp/util.buildTag "$(shell git describe --dirty)"
install: LDFLAGS += -X github.com/lampdb/lamp/util.buildTime "$(shell date -u '+%Y/%m/%d %H:%M:%S')"
install: LDFLAGS += -X github.com/lampdb/lamp/util.buildDeps "$(shell GOPATH=${GOPATH} build/depvers.sh)"
install:
	$(GO) install -tags '$(TAGS)' $(GOFLAGS) -ldflags '$(LDFLAGS)' -v

# Similar to "testrace", we want to cache the build before running the
# tests.
.PHONY: test
test:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -run $(TESTS) -cpu $(CPUS) $(PKG) -timeout $(TESTTIMEOUT) $(TESTFLAGS)

.PHONY: testslow
testslow: TESTFLAGS += -v
testslow:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -run $(TESTS) -cpu $(CPUS) $(PKG) -timeout $(TESTTIMEOUT) $(TESTFLAGS) | grep -F ': Test' | sed -E 's/(--- PASS: |\(|\))//g' | awk '{ print $$2, $$1 }' | sort -rn | head -n 10

.PHONY: testraceslow
testraceslow: TESTFLAGS += -v
testraceslow:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -race -run $(TESTS) -cpu $(CPUS) $(PKG) -timeout $(RACETIMEOUT) $(TESTFLAGS) | grep -F ': Test' | sed -E 's/(--- PASS: |\(|\))//g' | awk '{ print $$2, $$1 }' | sort -rn | head -n 10

# "go test -i" builds dependencies and installs them into GOPATH/pkg, but does not run the
# tests. Run it as a part of "testrace" since race-enabled builds are not covered by
# "make build", and so they would be built from scratch every time (including the
# slow-to-compile cgo packages).
.PHONY: testrace
testrace:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -race -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -race -run $(TESTS) -cpu $(CPUS) $(PKG) -timeout $(RACETIMEOUT) $(TESTFLAGS)

.PHONY: bench
bench:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -run $(TESTS) -cpu $(CPUS) -bench $(TESTS) $(PKG) -timeout $(BENCHTIMEOUT) $(TESTFLAGS)

.PHONY: coverage
coverage:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -cover -run $(TESTS) -cpu $(CPUS) $(PKG) $(TESTFLAGS)

.PHONY: acceptance
acceptance:
	@acceptance/run.sh

.PHONY: check
check:
	@echo "checking for tabs in shell scripts"
	@! git grep -F '	' -- '*.sh'
	@echo "checking for \"path\" imports"
	@! git grep -F '"path"' -- '*.go'
	@echo "errcheck"
	@! errcheck -ignore 'bytes:Write.*,io:(Close|Write),net:Close,net/http:(Close|Write),net/rpc:Close,os:Close,database/sql:Close,github.com/spf13/cobra:Usage' $(PKG) | grep -vE 'yacc\.go:'
	@echo "vet"
	@! go tool vet . 2>&1 | \
	  grep -vE '^vet: cannot process directory .git'
	@echo "vet --shadow"
	@! go tool vet --shadow . 2>&1 | \
	  grep -vE '(declaration of err shadows|^vet: cannot process directory \.git)'
	@echo "golint"
	@! golint $(PKG) | \
	  grep -vE '(\.pb\.go|embedded\.go|_string\.go|LastInsertId|sql/parser/(yaccpar|sql\.y):)' \
	  # https://golang.org/pkg/database/sql/driver/#Result :(
	@echo "gofmt (simplify)"
	@! gofmt -s -d -l . 2>&1 | grep -vE '^\.git/'
	@echo "goimports"
	@! goimports -l . | grep -vF 'No Exceptions'

.PHONY: clean
clean:
	$(GO) clean -tags '$(TAGS)' $(GOFLAGS) -i github.com/lampdb/...
	find . -name '*.test' -type f -exec rm -f {} \;
	rm -rf build/deploy/build

# Store all of the dependencies which are not part of the standard
# library or lampdb/lamp in build/devbase/deps
.PHONY: storedeps
storedeps:
	go list -f '{{range .Deps}}{{printf "%s\n" .}}{{end}}' ./... | sort | uniq | \
	 grep -E '[^/]+\.[^/]+/' | grep -vF 'github.com/lampdb/lamp' > build/devbase/deps

GITHOOKS := $(subst githooks/,.git/hooks/,$(wildcard githooks/*))
.git/hooks/%: githooks/%
	@echo installing $<
	@rm -f $@
	@mkdir -p $(dir $@)
	@ln -s ../../$(basename $<) $(dir $@)

# Update the git hooks and run the bootstrap script whenever any
# of them (or their dependencies) change.
.bootstrap: $(GITHOOKS) build/devbase/deps.sh GLOCKFILE
	@build/devbase/deps.sh
	@touch $@

-include .bootstrap
