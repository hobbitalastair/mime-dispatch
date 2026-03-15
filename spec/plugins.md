# Plugin Structure

To support a new mime type, helper binaries must be placed in mime type specific directories.

The supported helpers are:

- `metadata-list <file>`: Extracts metadata from file contents.
- `metadata-add <file> <key> <value>`: Adds metadata to file contents (appends to existing values). Optional.
- `metadata-delete <file> <key> <value>`: Deletes metadata from file contents. Optional.

Output is YAML-like, same as the main executable.

For standardized metadata keys and value formats used across plugins, see `spec/tags.md`.

These directories are searched in the following order (first match wins):

1. `$XDG_CONFIG_HOME/mimetype/<mime-type>/` (user)
2. `/etc/mimetype/<mime-type>/` (admin)
3. `/usr/lib/mimetype/<mime-type>/` (distro)

For example, the `audio/mpeg` mime type may be supported by binaries or symlinks to binaries like this:

```
/usr/lib/mimetype/audio/mpeg/metadata-list
/usr/lib/mimetype/audio/mpeg/metadata-add
/usr/lib/mimetype/audio/mpeg/metadata-delete
```

If a `metadata-list` binary is placed into `$XDG_CONFIG_HOME/mimetype/audio/mpeg/` it will be used in preference to the system one in `/usr/lib/mimetype`.
