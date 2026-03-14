# Extended Attributes

The tool manages the following xattr namespaces:

- `user.mime_type` - Stores the MIME type of the file
- `user.metadata.<key>` - Stores user-defined metadata keys

When listing metadata, xattrs outside these namespaces are ignored.