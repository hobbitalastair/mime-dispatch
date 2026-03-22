# Extended Attributes

The tool manages the following xattr namespaces:

- `user.mime_type` - Stores the MIME type of the file
- `user.metadata.<key>` - Stores user-defined metadata keys

xattrs outside these namespaces are ignored.

To support storing multiple different values for the same key, values are encoded in CSV format (RFC 4180). Values containing commas, double quotes, or newlines are enclosed in double quotes, and double quotes within values are escaped by doubling them.
