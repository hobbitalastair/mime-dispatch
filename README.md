# mime-dispatch

A suite of file-type-independent tools dispatched by MIME type. Plugins register handlers for specific MIME types and commands; the dispatcher selects the right handler based on each file's detected type.

## Tools

- `metadata` -- List, add, and delete file metadata. Combines plugin output with extended attributes (xattrs) as a secondary storage layer for file formats that don't support embedded metadata.
- `open` -- Open files using MIME-type-specific handlers.
- `mime-dispatch-install` -- Create symlink structures to install plugins.

## Plugins

Included plugins:

- `yaml-frontmatter` -- YAML front matter in Markdown and plain text files (read/write).
- `audio` -- MP3 (ID3), OGG (Vorbis), and FLAC tags (read-only).
- `audio-mutagen` -- MP3, OGG, and FLAC tag (write).
- `image` -- JPEG EXIF data (read-only).

See `spec/plugins.md` for how to write and install plugins.

## Dependencies

- Go 1.25+
- `perl-file-mimeinfo` (provides the `mimetype` command for MIME type detection)
- `python` and `python-mutagen` for audio tag writing

## Building

```sh
make build
```

Binaries are placed in `build/` by default. Override with `OUTDIR`:

```sh
make build OUTDIR=/usr/local/bin
```

## Testing

```sh
make test          # unit + end-to-end
make test-unit     # unit tests only
make test-e2e      # end-to-end (builds all binaries, installs plugins)
```

## Specifications

- `spec/cli.md` -- CLI interface and flags.
- `spec/flow.md` -- Metadata and open command execution flow.
- `spec/plugins.md` -- Plugin structure, search paths, and installation.
- `spec/tags.md` -- Standardized metadata tag names and value formats.
- `spec/xattr.md` -- Extended attribute namespaces and encoding.
