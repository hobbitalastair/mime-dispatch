# Metadata

If metadata for files is stored inside the files, I need a way to programmatically extract the data.
This should be plug-in based to support new file formats.

Metadata can be assumed to be stored as key:value pairs, where both are UTF-8 encoded strings?
Output should be in some standard format (eg YAML?) although this may add serialization/deserialization requirements.

There should be a command line tool to dump the metadata from a file.
It would be smart to have a library that loaded the config up front to avoid the execution + config load cost when processing many files.

It would be nice to be able to manipulate the metadata (delete and set).

Most metadata should live inside the file, but some file formats do not support this. It might be wise to use extended attributes as a secondary location.
The file's mime type should likely be exposed as metadata, but stored as an extended attribute.

