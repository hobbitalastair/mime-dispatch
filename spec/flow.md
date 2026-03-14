# Overall Flow

## 1. Read mime type from xattr

Attempt to read `user.mime_type` extended attribute. If present, use this mime type.

## 2. Detect mime type if not in xattr

If the xattr is not present, run the `mimetype` command on the file.

If successful, store the result in `user.mime_type` xattr for future use.

If the command fails, print an error to stderr and exit with a non-zero exit code.

## 3. Find plugin

Search plugin directories in the following order (first match wins):

1. `$XDG_CONFIG_HOME/metadata/plugins/<mime-type>/`
2. `/etc/metadata/plugins/<mime-type>/`
3. `/usr/lib/metadata/plugins/<mime-type>/`

If no plugin is found, print an error to stderr and exit with a non-zero exit code.

## 4. Execute plugin

The plugin is executed as a subprocess. It communicates with the main program via stdin/stdout.

The plugin only considers file contents and ignores the file's extended attributes.

## 5. Merge metadata

Metadata from both file contents and extended attributes are combined:

- Xattr takes precedence on key conflicts
- Multi-valued keys are merged

The merged result is returned.
