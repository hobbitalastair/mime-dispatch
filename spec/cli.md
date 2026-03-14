# CLI Interface

## Common Flags

- `--xattr-only` - Only access extended attributes
- `--file-only` - Only access file contents

By default, commands consider both file contents and extended attributes. Extended attributes take precedence in case of key conflicts.

## Commands

### list
List all metadata from a file.

```
metadata list <file>
```

### set
Set a metadata key/value pair. If the key exists, it is replaced in its current location. If it exists in both locations, only xattr is replaced. If it doesn't exist, it is added to file contents.

```
metadata set <file> <key> <value>
```

### delete
Delete a metadata key from all locations where it exists.

```
metadata delete <file> <key>
```

## Output Format

All commands output flat YAML. Keys with multiple values use YAML sequences:

```
key: value
multi-valued:
  - value1
  - value2
another_key: another_value
```
