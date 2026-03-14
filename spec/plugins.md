# Plugin Structure

## Plugin Search Paths

Plugins are discovered by searching directories in the following order (first match wins):

1. `$XDG_CONFIG_HOME/metadata/plugins/<mime-type>/` (user)
2. `/etc/metadata/plugins/<mime-type>/` (admin)
3. `/usr/lib/metadata/plugins/<mime-type>/` (distro)

This allows users to override system-provided plugins.

## Plugin Directory Structure

Each mime type has its own directory containing a symlink to the plugin executable:

```
/usr/lib/metadata/plugins/image/jpeg/metadata
/etc/metadata/plugins/text/markdown/metadata
$XDG_CONFIG_HOME/metadata/plugins/audio/mp3/metadata
```

The symlink target is managed by the package manager.

## Plugin executable CLI

The plugin should use the same CLI format as the main executable, except it should only consider the file contents and ignore the file's extended attributes.

