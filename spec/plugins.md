# Plugin Structure

## Plugin Search Paths

Plugins are discovered by searching directories in the following order (first match wins):

1. `$XDG_CONFIG_HOME/metadata/plugins/<mime-type>/` (user)
2. `/etc/metadata/plugins/<mime-type>/` (admin)
3. `/usr/lib/metadata/plugins/<mime-type>/` (distro)

This allows users to override system-provided plugins.

## Plugin Directory Structure

Each mime type has its own directory (using the full mime type with slashes) containing a symlink to the plugin executable:

```
/usr/lib/metadata/plugins/text/markdown/metadata
/etc/metadata/plugins/text/markdown/metadata
$XDG_CONFIG_HOME/metadata/plugins/text/markdown/metadata
```

The symlink target is managed by the package manager.

## Plugin executable CLI

The plugin uses the same CLI interface as the main executable:

```
<plugin-name> list <file>
<plugin-name> set <file> <key> <value>
<plugin-name> delete <file> <key>
```

The plugin only considers file contents and ignores the file's extended attributes.

Output is flat YAML key-value pairs, same as the main executable.

