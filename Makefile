# lazy-notes — Make drives Homebrew (deps, local tap, services).

BREW    ?= brew
TAP     := jborkowski/lazy-notes
FORMULA := $(TAP)/lazy-notes
export HOMEBREW_NO_AUTO_UPDATE ?= 1

.PHONY: all help deps tap pack install setup start stop restart status sync logs uninstall

all: install

help:
	@echo "make deps       brew install go duckdb ffmpeg"
	@echo "make install    pack sources into tap + brew install"
	@echo "make setup      lazy-notes setup"
	@echo "make start|stop|restart"
	@echo "make status|sync|logs"
	@echo "make uninstall"

deps:
	$(BREW) install go duckdb ffmpeg
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
	lazy-notes setup

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
