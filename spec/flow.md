# Overall Flow

## 1. Read mime type from xattr

Attempt to read `user.mime_type` extended attribute. If present, use this mime type.

## 2. Detect mime type if not in xattr

If the xattr is not present, run the `mimetype` command on the file.

If successful, store the result in `user.mime_type` xattr for future use.

If the command fails, print an error to stderr and exit with a non-zero exit code.

## 3. Find plugin

Search for command-specific plugin:

1. `$XDG_CONFIG_HOME/metadata/plugins/<mime-type>/<command>`
2. `/etc/metadata/plugins/<mime-type>/<command>`
3. `/usr/lib/metadata/plugins/<mime-type>/<command>`

Where `<command>` is `list`, `add`, or `delete`.

If using --file-only and no plugin is found:
- For `list`: Print warning to stderr and return empty metadata.
- For `add`: Print a warning to stderr and exit with a non-zero exit code.
- For `delete`: If the list plugin exists and the key exists in file metadata, print a warning to stderr and exit with a non-zero exit code. Otherwise, continue (xattr deleted separately).

If using the default (both file and xattrs) and no plugin is found:
- For `list`: Print warning to stderr and return just the xattr metadata.
- For `add`: Set only in xattrs.
- For `delete`: Delete only in xattrs. If the list plugin exists and the key exists in file metadata, print a warning to stderr and exit with a non-zero exit code.

## 4. Execute plugin

The plugin is executed as a subprocess. It communicates with the main program via stdin/stdout.

The plugin only considers file contents and ignores the file's extended attributes.

## 5. Merge metadata

Metadata from both file contents and extended attributes are combined. Keys may be multi valued, but duplicates (the same entry in xattr and in the file) should be removed.

The merged result is returned.
