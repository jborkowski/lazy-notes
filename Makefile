# lazy-notes — Make drives Homebrew (deps, local tap, services).

BREW    ?= brew
TAP     := jborkowski/lazy-notes
FORMULA := $(TAP)/lazy-notes
export HOMEBREW_NO_AUTO_UPDATE ?= 1

# Release knobs: make release VERSION=0.1.3
VERSION ?=
ARCHS   ?= arm64
RELEASE_FLAGS ?=

.PHONY: all help deps tap pack install setup start stop restart status sync logs uninstall jscpd test \
	release release-dry dist-arm64

all: install

help:
	@echo "make deps       brew tap/install memo + gogcli + hf go duckdb ffmpeg"
	@echo "make install    pack sources into tap + brew install"
	@echo "make setup      lazy-notes onboard (step-by-step + doctor)"
	@echo "make start|stop|restart"
	@echo "make status|sync|logs"
	@echo "make test       go test ./..."
	@echo "make jscpd      copy/paste detector (npx jscpd)"
	@echo "make uninstall"
	@echo "make release VERSION=X.Y.Z   bump/tag/upload/pin"
	@echo "make release-dry VERSION=X.Y.Z"
	@echo "make dist-arm64 VERSION=X.Y.Z   build tarball only → dist/"

test:
	go test ./...

jscpd:
	npx --yes jscpd . --config .jscpd.json

# If `brew install lazy-notes` fails to resolve memo/gogcli (tap-prefixed depends_on),
# run `make deps` first — it taps those formulas and installs them explicitly.
deps:
	$(BREW) tap antoniorodr/memo
	$(BREW) install antoniorodr/memo/memo
	$(BREW) tap openclaw/tap
	$(BREW) install openclaw/tap/gogcli
	$(BREW) install hf go duckdb ffmpeg
	@if ! $(BREW) list --cask superwhisper >/dev/null 2>&1 && [ ! -d /Applications/superwhisper.app ]; then \
		$(BREW) install --cask superwhisper; \
	else \
		echo "superwhisper.app already present"; \
	fi

tap:
	@if ! $(BREW) tap | grep -qx "$(TAP)"; then \
		$(BREW) tap-new "$(TAP)" --branch main; \
	fi

pack: tap
	@TAPDIR="$$($(BREW) --repo $(TAP))"; \
	mkdir -p "$$TAPDIR/Formula"; \
	rm -rf "$$TAPDIR/build-src" "$$TAPDIR/lazy-notes-src.tar.gz"; \
	rsync -a \
		--exclude '.git/' \
		--exclude 'bin/' \
		--exclude '.cursor/' \
		--exclude '*.sqlite' \
		--exclude '.DS_Store' \
		./ "$$TAPDIR/build-src/"; \
	tar -C "$$TAPDIR" -czf "$$TAPDIR/lazy-notes-src.tar.gz" build-src; \
	cp -f Formula/lazy-notes.rb "$$TAPDIR/Formula/lazy-notes.rb"; \
	echo "packed $$TAPDIR/lazy-notes-src.tar.gz"

install: deps pack
	@if $(BREW) list --formula "$(FORMULA)" >/dev/null 2>&1; then \
		$(BREW) reinstall --build-from-source "$(FORMULA)"; \
	else \
		$(BREW) install --build-from-source "$(FORMULA)"; \
	fi

setup: install
	lazy-notes onboard

start:
	$(BREW) services start $(FORMULA)

stop:
	$(BREW) services stop $(FORMULA)

restart:
	$(BREW) services restart $(FORMULA)

status:
	-$(BREW) services info $(FORMULA)
	-lazy-notes status

sync:
	lazy-notes sync

logs:
	@prefix="$$($(BREW) --prefix $(FORMULA))"; \
	tail -n 80 -f "$$prefix/var/log/lazy-notes.log" "$$prefix/var/log/lazy-notes.err.log"

uninstall:
	-$(BREW) services stop $(FORMULA)
	-$(BREW) uninstall --force $(FORMULA)
	-$(BREW) untap $(TAP)

# Codified release only: Formula bump → tag → GitHub darwin archives → pin.
# Example: make release VERSION=0.1.3
# Optional: ARCHS=arm64,amd64,universal RELEASE_FLAGS='--notes "…"'
release:
	@test -n "$(VERSION)" || (echo "usage: make release VERSION=X.Y.Z"; exit 2)
	./scripts/release.sh "$(VERSION)" --archs "$(ARCHS)" $(RELEASE_FLAGS)

release-dry:
	@test -n "$(VERSION)" || (echo "usage: make release-dry VERSION=X.Y.Z"; exit 2)
	./scripts/release.sh "$(VERSION)" --archs "$(ARCHS)" --dry-run $(RELEASE_FLAGS)

# Build only (no git/gh). Useful to inspect the arm64 artifact layout.
dist-arm64:
	@test -n "$(VERSION)" || (echo "usage: make dist-arm64 VERSION=X.Y.Z"; exit 2)
	@V="$(VERSION)"; V="$${V#v}"; TAG="v$$V"; \
	NAME="lazy-notes-$${TAG}-darwin-arm64"; \
	rm -rf "dist/$$NAME" "dist/$${NAME}.tar.gz"; mkdir -p "dist/$$NAME/config"; \
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$$V" -o "dist/$$NAME/lazy-notes" ./cmd/lazy-notes; \
	/bin/cp README.md "dist/$$NAME/"; \
	rsync -a config/ "dist/$$NAME/config/"; \
	tar -C dist -czf "dist/$${NAME}.tar.gz" "$$NAME"; \
	(cd dist && shasum -a 256 "$${NAME}.tar.gz" > checksums-arm64.txt); \
	file "dist/$$NAME/lazy-notes"; \
	echo "wrote dist/$${NAME}.tar.gz"
