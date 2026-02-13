CONFIGDIR     = $(HOME)/.config/elephant
PROVIDERDIR   = $(CONFIGDIR)/providers
ELEPHANT_SRC  = /tmp/elephant-src
PLUGIN_DIR    = $(ELEPHANT_SRC)/internal/providers/remindy
PLUGIN_NAME   = remindy.so
ELEPHANT_VERSION := $(shell elephant version 2>/dev/null)
VERSION_FILE  = $(ELEPHANT_SRC)/.remindy-version

GO_BUILD_FLAGS = -buildvcs=false -buildmode=plugin -trimpath
export GOEXPERIMENT = nodwarf5

.PHONY: all build install uninstall clone clean clean-elephant clean-all help

all: build

clone:
	@command -v elephant >/dev/null 2>&1 || { echo "Error: elephant not in PATH"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "Error: go not in PATH"; exit 1; }
	@if [ -z "$(ELEPHANT_VERSION)" ]; then echo "Error: could not detect elephant version"; exit 1; fi
	@if [ -d "$(ELEPHANT_SRC)" ]; then \
		if [ ! -f "$(VERSION_FILE)" ] || [ "$$(cat $(VERSION_FILE))" != "$(ELEPHANT_VERSION)" ]; then \
			echo "Version mismatch (want v$(ELEPHANT_VERSION), have $$(cat $(VERSION_FILE) 2>/dev/null || echo unknown))"; \
			echo "Run: make clean-elephant && make"; \
			exit 1; \
		fi; \
		echo "elephant v$(ELEPHANT_VERSION) source ready"; \
	else \
		echo "Cloning elephant v$(ELEPHANT_VERSION)..."; \
		git clone --depth 1 --branch "v$(ELEPHANT_VERSION)" \
			https://github.com/abenz1267/elephant.git "$(ELEPHANT_SRC)" && \
		echo "$(ELEPHANT_VERSION)" > "$(VERSION_FILE)"; \
	fi

build: clone
	mkdir -p $(PLUGIN_DIR)
	cp setup.go $(PLUGIN_DIR)/setup.go
	cd $(PLUGIN_DIR) && go build -o $(PLUGIN_NAME) $(GO_BUILD_FLAGS)
	cp $(PLUGIN_DIR)/$(PLUGIN_NAME) ./$(PLUGIN_NAME)
	@echo "Built $(PLUGIN_NAME) for elephant v$(ELEPHANT_VERSION)"

install: build
	mkdir -p $(PROVIDERDIR)
	cp $(PLUGIN_NAME) $(PROVIDERDIR)/$(PLUGIN_NAME)
	cp ../remindy.toml $(CONFIGDIR)/remindy.toml
	systemctl --user restart elephant 2>/dev/null || true
	@echo "Installed to $(PROVIDERDIR)"

uninstall:
	rm -f $(PROVIDERDIR)/$(PLUGIN_NAME)
	rm -f $(CONFIGDIR)/remindy.toml

clean:
	rm -f $(PLUGIN_NAME)
	rm -rf $(PLUGIN_DIR)

clean-elephant:
	rm -rf $(ELEPHANT_SRC)

clean-all: clean clean-elephant

help:
	@echo "Targets:"
	@echo "  make           Build plugin (auto-clones elephant source)"
	@echo "  make install   Build + install to ~/.config/elephant/"
	@echo "  make uninstall Remove plugin and config"
	@echo "  make clean     Remove local build artifacts"
	@echo "  make clean-elephant  Remove elephant source (/tmp/elephant-src)"
	@echo "  make clean-all       Remove everything"
	@echo ""
	@echo "Detected elephant: v$(ELEPHANT_VERSION)"
