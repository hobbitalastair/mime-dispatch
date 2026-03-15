package lib

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
)

type Options struct {
	XattrOnly bool
	FileOnly  bool
}

type Metadata map[string][]string

func MergeMetadata(fileMeta, xattrMeta map[string][]string) map[string][]string {
	seen := make(map[string]map[string]struct{})
	addMetadata := func(key string, values []string) {
		if _, exists := seen[key]; !exists {
			seen[key] = make(map[string]struct{})
		}
		for _, value := range values {
			seen[key][value] = struct{}{}
		}
	}

	for key, values := range fileMeta {
		addMetadata(key, values)
	}
	for key, values := range xattrMeta {
		addMetadata(key, values)
	}

	result := make(map[string][]string)
	for key, values := range seen {
		result[key] = slices.Sorted(maps.Keys(values))
	}

	return result
}

func GetMetadata(filePath string, opts Options) (Metadata, error) {
	var xattrMeta Metadata
	var fileMeta Metadata

	if !opts.FileOnly {
		xattrs, err := GetXattr(filePath)
		if err != nil && err != ErrXattrNotSupported {
			return nil, err
		}
		if err == nil {
			xattrMeta = xattrs
		}
	}

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return nil, err
		}

		if !opts.FileOnly {
			if xattrMeta == nil {
				xattrMeta = make(Metadata)
			}
			xattrMeta["mime_type"] = []string{mimeType}
		}

		pluginPath, err := FindPluginForCommand(mimeType, PluginList)
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if errors.As(err, &noPluginErr) {
				fmt.Fprintf(os.Stderr, "Warning: no list plugin found for mime type %s, returning empty metadata\n", mimeType)
			} else {
				return nil, err
			}
		} else {
			pluginMeta, err := RunPlugin(pluginPath, PluginList, filePath, "", "")
			if err != nil {
				return nil, err
			}
			fileMeta = pluginMeta
		}
	}

	return MergeMetadata(fileMeta, xattrMeta), nil
}

func AddMetadata(filePath, key, value string, opts Options) error {
	if opts.XattrOnly {
		return addToXattr(filePath, key, value)
	}

	mimeType, err := getMimeType(filePath, opts)
	if err != nil {
		return err
	}

	pluginPath, err := FindPluginForCommand(mimeType, PluginAdd)
	if err == nil {
		_, err = RunPlugin(pluginPath, PluginAdd, filePath, key, value)
		return err
	}

	var noPluginErr ErrNoPluginFound
	if !errors.As(err, &noPluginErr) {
		return err
	}

	if opts.FileOnly {
		return fmt.Errorf("cannot write to file: no add plugin found for mime type %s", mimeType)
	}

	return addToXattr(filePath, key, value)
}

// addToXattr appends a value to an xattr key (or creates it if it doesn't exist)
func addToXattr(filePath, key, value string) error {
	// Get current values
	currentValue, err := GetXattrValue(filePath, key)
	if err != nil && err != ErrXattrNotSupported {
		return err
	}

	// Decode current values
	values := decodeCSV(currentValue)

	// Append new value
	values = append(values, value)

	// Encode and set
	encoded := encodeCSV(values)
	return SetXattr(filePath, key, encoded)
}

func DeleteMetadata(filePath, key, value string, opts Options) error {
	var mimeType string
	var deletePluginPath string

	if !opts.XattrOnly {
		var err error
		mimeType, err = getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPluginForCommand(mimeType, PluginDelete)
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if !errors.As(err, &noPluginErr) {
				return err
			}

			// No delete plugin -- check if the key exists in the file via the list plugin,
			// because if it does we can't delete it.
			listPluginPath, err := FindPluginForCommand(mimeType, PluginList)
			if err == nil {
				pluginMeta, err := RunPlugin(listPluginPath, PluginList, filePath, "", "")
				if err != nil {
					return err
				}
				if _, keyExistsInFile := pluginMeta[key]; keyExistsInFile {
					return fmt.Errorf("cannot delete key %q from file: file is read-only", key)
				}
			}
		} else {
			deletePluginPath = pluginPath
		}
	}

	if !opts.FileOnly {
		if err := DeleteXattr(filePath, key, value); err != nil {
			return err
		}
	}

	if deletePluginPath != "" {
		_, err := RunPlugin(deletePluginPath, PluginDelete, filePath, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func getMimeType(filePath string, opts Options) (string, error) {
	if opts.FileOnly {
		return DetectMimetype(filePath)
	}

	mimeType, err := GetXattrValue(filePath, "mime_type")
	if err != nil {
		return "", err
	}

	if mimeType != "" {
		return mimeType, nil
	}

	mimeType, err = DetectMimetype(filePath)
	if err != nil {
		return "", err
	}

	if err := SetXattr(filePath, "mime_type", mimeType); err != nil {
		return "", err
	}

	return mimeType, nil
}
