# Standard Metadata Tags

This document defines standard metadata tags and value formats used by the metadata tool.

Values are serialized as YAML scalars or YAML sequences of scalars.

## Core Tags

- `mime_type`: MIME type string (for example `text/markdown`, `image/jpeg`).
- `datetime`: ISO 8601 style datetime. Reduced precision is allowed (for example `YYYY`, `YYYY-MM`, `YYYY-MM-DD`, or `YYYY-MM-DDThh:mm:ss`). Include timezone offset in the same field when available (for example `2026-03-15T14:30:45+01:00`).
- `location`: Decimal latitude and longitude as `latitude,longitude` (for example `51.5074,-0.1278`).
- `comment`: Free-form text comment.

## Audio Tags

- `title`: Track title.
- `album`: Album title.
- `artist`: Track artist.
- `album_artist`: Album artist.
- `composer`: Composer.
- `genre`: Genre label.
- `year`: Four-digit year string.

Additional plugin-specific tags may exist, but the tags above are the standardized keys.
