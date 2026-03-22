# Overall Flow

## 1. Read mime type from xattr

Attempt to read `user.mime_type` extended attribute. If present, use this mime type.

## 2. Detect mime type if not in xattr

If the xattr is not present, run the `mimetype` command on the file.

If successful, store the result in `user.mime_type` xattr for future use.

If the command fails, print an error to stderr and exit with a non-zero exit code.

## 3. Find plugin

Search for command-specific plugin, where `<command>` is `metadata-list`, `metadata-add`, or `metadata-delete`.

If using --file-only and no plugin is found:
- For `metadata-list`: Print warning to stderr and return empty metadata.
- For `metadata-add`: Print a warning to stderr and exit with a non-zero exit code.
- For `metadata-delete`: If the list plugin exists and the key exists in file metadata, print a warning to stderr and exit with a non-zero exit code. Otherwise, continue (xattr deleted separately).

If using the default (both file and xattrs) and no plugin is found:
- For `metadata-list`: Print warning to stderr and return just the xattr metadata.
- For `metadata-add`: Set only in xattrs.
- For `metadata-delete`: Delete only in xattrs. If the list plugin exists and the key exists in file metadata, print a warning to stderr and exit with a non-zero exit code.

## 4. Execute plugin

The plugin is executed as a subprocess. It communicates with the main program via stdin/stdout.
It only considers file contents and ignores the file's extended attributes.

## 5. Merge metadata

Metadata from both file contents and extended attributes are combined. Keys may be multi valued, but duplicates (the same entry in xattr and in the file) should be removed.

The merged result is returned.

## Open Command Flow

The `open` binary follows a simpler flow for each file:

### 1. Detect MIME type

Same as steps 1–2 above: read `user.mime_type` from xattr, falling back to detection via the `mimetype` command.

### 2. Find open handler

Search for `<mime-type>/open` in the plugin search directories (same order as metadata plugins).

If no handler is found, print an error for that file and continue to the next file.

### 3. Execute handler

Run the handler with the file path as its sole argument. The handler inherits stdin, stdout, and stderr so it can be interactive (e.g. launching an editor, viewer, or player). No sandboxing is applied.

If the handler exits with a non-zero status, the error is reported and processing continues with the remaining files. The `open` binary exits non-zero if any file failed.
