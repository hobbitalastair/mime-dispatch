# CLI Interface

## Common Flags

- `--xattr-only` - Only access extended attributes
- `--file-only` - Only access file contents

Commands apply to both file contents and extended attributes.

## Commands

### list
List all metadata from a file.

```
metadata list <file>
```

### add
Add a metadata key/value pair (appends to existing values).

```
metadata add <file> <key> <value>
```

### delete
Delete a specific metadata key/value pair.

```
metadata delete <file> <key> <value>
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

YAML formatting details (quoting style, scalar style, and indentation) are produced by the YAML serializer and may vary; consumers should parse YAML rather than rely on exact whitespace.

For standardized metadata keys and value formats, see `spec/tags.md`.
