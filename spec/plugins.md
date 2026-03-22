# Plugin Structure

To support a new mime type, helper binaries must be placed in mime type specific directories.

The supported helpers are:

- `metadata-list <file>`: Extracts metadata from file contents.
- `metadata-add <file> <key> <value>`: Adds metadata to file contents (appends to existing values). Optional.
- `metadata-delete <file> <key> <value>`: Deletes metadata from file contents. Optional.
- `open <file>`: Opens a file in an appropriate application. Unlike metadata plugins, open handlers inherit stdin/stdout/stderr and are not sandboxed. Optional.

Output for metadata helpers is YAML-like, same as the main executable. Open handlers produce no structured output.

For standardized metadata keys and value formats used across plugins, see `spec/tags.md`.

These directories are searched in the following order (first match wins):

1. `$XDG_CONFIG_HOME/mimetype/<mime-type>/` (user)
2. `/etc/mimetype/<mime-type>/` (system)
3. `/usr/lib/mimetype/<mime-type>/` (vendor)

For example, the `audio/mpeg` mime type may be supported by binaries or symlinks to binaries like this:

```
/usr/lib/mimetype/audio/mpeg/metadata-list
/usr/lib/mimetype/audio/mpeg/metadata-add
/usr/lib/mimetype/audio/mpeg/metadata-delete
/usr/lib/mimetype/audio/mpeg/open
```

If a `metadata-list` binary is placed into `$XDG_CONFIG_HOME/mimetype/audio/mpeg/` it will be used in preference to the system one in `/usr/lib/mimetype`.

## Plugin Capabilities

Plugins declare which MIME types and commands they support via the `--capabilities` flag. The output is YAML:

```yaml
mimetypes:
    - audio/mpeg
    - audio/ogg
    - audio/x-vorbis+ogg
commands:
    - metadata-list
    - open
```

## Installing Plugins

The `mime-dispatch-install` tool creates (or removes) the symlink structure for a plugin binary based on its declared capabilities.

```
mime-dispatch-install [--user|--system|--vendor] [--mimetype <type>]... [--uninstall] <binary-path>
```

### Levels

| Flag | Directory | Purpose |
|------|-----------|---------|
| `--user` | `$XDG_CONFIG_HOME/mimetype/` | Current user overrides |
| `--system` | `/etc/mimetype/` | System administrator configuration |
| `--vendor` | `/usr/lib/mimetype/` | Distribution package defaults |

### Examples

Install a plugin for the current user:

```
mime-dispatch-install --user ~/.local/lib/mime-dispatch/metadata-yaml-frontmatter
```

Install for a MIME type the plugin doesn't explicitly declare:

```
mime-dispatch-install --user --mimetype text/x-rst ~/.local/lib/mime-dispatch/metadata-yaml-frontmatter
```

Uninstall a plugin:

```
mime-dispatch-install --user --uninstall ~/.local/lib/mime-dispatch/metadata-yaml-frontmatter
```

Distribution packaging (with DESTDIR):

```
DESTDIR="$pkgdir" mime-dispatch-install --vendor "$pkgdir/usr/lib/mime-dispatch/metadata-audio"
```

The `DESTDIR` environment variable prefixes all filesystem operations. Symlinks always point to the final absolute path (with DESTDIR stripped).
