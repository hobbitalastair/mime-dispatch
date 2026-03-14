# Plugin Structure

## Plugin Search Paths

Plugins are discovered by searching directories in the following order (first match wins):

1. `$XDG_CONFIG_HOME/metadata/plugins/<mime-type>/` (user)
2. `/etc/metadata/plugins/<mime-type>/` (admin)
3. `/usr/lib/metadata/plugins/<mime-type>/` (distro)

This allows users to override system-provided plugins.

## Plugin Directory Structure

Each mime type has its own directory containing command-specific symlinks:

```
/usr/lib/metadata/plugins/audio/mpeg/list
/usr/lib/metadata/plugins/audio/mpeg/add
/usr/lib/metadata/plugins/audio/mpeg/delete
```

Supported commands: `list`, `add`, `delete`. `list` is not optional. `add` and `delete` are optional.

- `list`: Extracts metadata from file contents.
- `add`: Adds metadata to file contents (appends to existing values).
- `delete`: Deletes metadata from file contents.

The symlink target is managed by the package manager.

## Plugin executable CLI

The plugin uses the same CLI interface as the main executable:

```
list <file>
add <file> <key> <value>
delete <file> <key> <value>
```

The plugin only considers file contents and ignores the file's extended attributes.

Output is YAML-like, same as the main executable.
