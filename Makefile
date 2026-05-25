OUTDIR ?= build
OUTDIR := $(abspath $(OUTDIR))
GOFLAGS ?=
LDFLAGS ?=

GO_BUILD = go build $(GOFLAGS)
ifdef LDFLAGS
GO_BUILD += -ldflags "$(LDFLAGS)"
endif

BINARIES = metadata open mime-dispatch-install
PLUGINS = metadata-yaml-frontmatter metadata-audio metadata-image metadata-audio-write

.PHONY: build binaries plugins test test-unit test-e2e vet format clean

build: binaries plugins

binaries: $(addprefix $(OUTDIR)/,$(BINARIES))

plugins: $(addprefix $(OUTDIR)/,$(PLUGINS))

$(OUTDIR)/metadata:
	$(GO_BUILD) -o $@ ./cmd/metadata

$(OUTDIR)/open:
	$(GO_BUILD) -o $@ ./cmd/open

$(OUTDIR)/mime-dispatch-install:
	$(GO_BUILD) -o $@ ./cmd/mime-dispatch-install

$(OUTDIR)/metadata-yaml-frontmatter:
	cd plugins/yaml-frontmatter && $(GO_BUILD) -o $@ .

$(OUTDIR)/metadata-audio:
	cd plugins/audio && $(GO_BUILD) -o $@ .

$(OUTDIR)/metadata-image:
	cd plugins/image && $(GO_BUILD) -o $@ .

$(OUTDIR)/metadata-audio-write:
	cp plugins/audio-mutagen/plugin.py $@
	chmod +x $@

test: test-unit test-e2e

test-unit:
	go test ./lib/ ./pkg/pluginio/

test-e2e:
	go test ./e2e/

vet:
	go vet ./...

format:
	gofmt -w .
	cd plugins/audio && gofmt -w .
	cd plugins/image && gofmt -w .
	cd plugins/yaml-frontmatter && gofmt -w .
	black plugins/audio-mutagen/plugin.py

clean:
	rm -rf $(OUTDIR)
