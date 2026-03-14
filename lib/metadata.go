package lib

import (
	"errors"
	"fmt"
	"os"
)

type Options struct {
	XattrOnly bool
	FileOnly  bool
}

type Metadata map[string][]string

func (m Metadata) ToYAML() string {
	result := ""
	for k, values := range m {
		if len(values) == 1 {
			result += k + ": " + values[0] + "\n"
		} else {
			result += k + ":\n"
			for _, v := range values {
				result += "  - " + v + "\n"
			}
		}
	}
	return result
}

func GetMetadata(filePath string, opts Options) (Metadata, error) {
	result := make(Metadata)

	xattrMeta := make(Metadata)
	fileMeta := make(Metadata)

	if !opts.FileOnly {
		xattrs, err := GetXattr(filePath)
		if err != nil && err != ErrXattrNotSupported {
			return nil, err
		}
		if err == nil {
			for k, v := range xattrs {
				xattrMeta[k] = v
			}
		}
	}

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return nil, err
		}

		if !opts.FileOnly {
			xattrMeta["mime_type"] = []string{mimeType}
		}

		pluginPath, err := FindPluginForCommand(mimeType, "list")
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if errors.As(err, &noPluginErr) {
				fmt.Fprintf(os.Stderr, "Warning: no list plugin found for mime type %s, returning empty metadata\n", mimeType)
			} else {
				return nil, err
			}
		} else {
			pluginMeta, err := RunPlugin(pluginPath, "list", filePath, "", "")
			if err != nil {
				return nil, err
			}

			for k, v := range pluginMeta {
				fileMeta[k] = v
			}
		}
	}

	result = MergeMetadata(fileMeta, xattrMeta)

	return result, nil
}

func AddMetadata(filePath, key, value string, opts Options) error {
	keyExistedInXattr := false
	keyExistedInFile := false

	// First, check current state only if not explicitly specifying where to write
	if !opts.FileOnly && !opts.XattrOnly {
		existed, err := xattrExists(filePath, key)
		if err != nil && err != ErrXattrNotSupported {
			return err
		}
		keyExistedInXattr = existed

		// Check if key exists in file
		if !opts.FileOnly {
			mimeType, err := getMimeType(filePath, opts)
			if err == nil {
				pluginPath, err := FindPluginForCommand(mimeType, "list")
				if err == nil {
					pluginMeta, err := RunPlugin(pluginPath, "list", filePath, "", "")
					if err == nil {
						_, keyExistedInFile = pluginMeta[key]
					}
				}
			}
		}
	}

	// Determine where to write based on options and current state
	if opts.XattrOnly {
		// Explicitly xattr-only: add to xattr
		return addToXattr(filePath, key, value)
	} else if opts.FileOnly {
		// Explicitly file-only: add to file via plugin
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPluginForCommand(mimeType, "add")
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if errors.As(err, &noPluginErr) {
				// No add plugin, but file-only was requested - this is an error
				return fmt.Errorf("cannot write to file: no add plugin found for mime type %s", mimeType)
			}
			return err
		}

		_, err = RunPlugin(pluginPath, "add", filePath, key, value)
		if err != nil {
			return err
		}
	} else {
		// Default behavior: xattr takes precedence
		if keyExistedInXattr {
			// Key exists in xattr: add to xattr (xattr takes precedence)
			return addToXattr(filePath, key, value)
		} else if keyExistedInFile {
			// Key exists only in file: add to file
			mimeType, err := getMimeType(filePath, opts)
			if err != nil {
				return err
			}

			pluginPath, err := FindPluginForCommand(mimeType, "add")
			if err != nil {
				var noPluginErr ErrNoPluginFound
				if errors.As(err, &noPluginErr) {
					// No plugin, but key was in file before - try xattr
					return addToXattr(filePath, key, value)
				} else {
					return err
				}
			} else {
				_, err = RunPlugin(pluginPath, "add", filePath, key, value)
				if err != nil {
					return err
				}
			}
		} else {
			// Key doesn't exist anywhere: write to file (default) or xattr if no plugin
			mimeType, err := getMimeType(filePath, opts)
			if err != nil {
				return err
			}

			pluginPath, err := FindPluginForCommand(mimeType, "add")
			if err != nil {
				var noPluginErr ErrNoPluginFound
				if errors.As(err, &noPluginErr) {
					// No plugin: add to xattr (fallback)
					return addToXattr(filePath, key, value)
				} else {
					return err
				}
			} else {
				// Plugin exists: add to file
				_, err = RunPlugin(pluginPath, "add", filePath, key, value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
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

func xattrExists(path, key string) (bool, error) {
	value, err := GetXattrValue(path, key)
	if err != nil {
		if err == ErrXattrNotSupported {
			return false, nil
		}
		return false, err
	}
	return value != "", nil
}

func DeleteMetadata(filePath, key, value string, opts Options) error {
	checkKeyExistsInFile := false

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		_, err = FindPluginForCommand(mimeType, "delete")
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if errors.As(err, &noPluginErr) {
				checkKeyExistsInFile = true
			} else {
				return err
			}
		}
	}

	if checkKeyExistsInFile {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPluginForCommand(mimeType, "list")
		if err == nil {
			pluginMeta, err := RunPlugin(pluginPath, "list", filePath, "", "")
			if err != nil {
				return err
			}
			if _, keyExistsInFile := pluginMeta[key]; keyExistsInFile {
				return fmt.Errorf("cannot delete key %q from file: file is read-only", key)
			}
		}
	}

	if !opts.FileOnly {
		if err := DeleteXattr(filePath, key, value); err != nil {
			return err
		}
	}

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPluginForCommand(mimeType, "delete")
		if err != nil {
			var noPluginErr ErrNoPluginFound
			if errors.As(err, &noPluginErr) {
			} else {
				return err
			}
		} else {
			_, err = RunPlugin(pluginPath, "delete", filePath, key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getMimeType(filePath string, opts Options) (string, error) {
	if opts.FileOnly {
		mimeType, err := DetectMimetype(filePath)
		if err != nil {
			return "", err
		}
		return mimeType, nil
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
