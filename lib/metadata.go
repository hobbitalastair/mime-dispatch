package lib

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

		pluginPath, err := FindPlugin(mimeType)
		if err != nil {
			return nil, err
		}

		pluginMeta, err := RunPlugin(pluginPath, "list", filePath, "", "")
		if err != nil {
			return nil, err
		}

		for k, v := range pluginMeta {
			fileMeta[k] = v
		}
	}

	result = MergeMetadata(fileMeta, xattrMeta)

	return result, nil
}

func SetMetadata(filePath, key, value string, opts Options) error {
	keyExistedInXattr := false
	keyExistedInFile := false

	if !opts.FileOnly {
		existed, err := xattrExists(filePath, key)
		if err != nil && err != ErrXattrNotSupported {
			return err
		}
		keyExistedInXattr = existed
	}

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPlugin(mimeType)
		if err != nil {
			return err
		}

		pluginMeta, err := RunPlugin(pluginPath, "list", filePath, "", "")
		if err != nil {
			return err
		}
		_, keyExistedInFile = pluginMeta[key]
	}

	if opts.XattrOnly {
		if !opts.FileOnly {
			if err := SetXattr(filePath, key, value); err != nil {
				return err
			}
		}
	} else if opts.FileOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPlugin(mimeType)
		if err != nil {
			return err
		}

		_, err = RunPlugin(pluginPath, "set", filePath, key, value)
		if err != nil {
			return err
		}
	} else if keyExistedInXattr && !keyExistedInFile {
		if !opts.FileOnly {
			if err := SetXattr(filePath, key, value); err != nil {
				return err
			}
		}
	} else if keyExistedInFile && !keyExistedInXattr {
		if !opts.XattrOnly {
			mimeType, err := getMimeType(filePath, opts)
			if err != nil {
				return err
			}

			pluginPath, err := FindPlugin(mimeType)
			if err != nil {
				return err
			}

			_, err = RunPlugin(pluginPath, "set", filePath, key, value)
			if err != nil {
				return err
			}
		}
	} else if keyExistedInXattr && keyExistedInFile {
		if !opts.FileOnly {
			if err := SetXattr(filePath, key, value); err != nil {
				return err
			}
		}
	} else if !keyExistedInXattr && !keyExistedInFile {
		if !opts.XattrOnly {
			mimeType, err := getMimeType(filePath, opts)
			if err != nil {
				return err
			}

			pluginPath, err := FindPlugin(mimeType)
			if err != nil {
				return err
			}

			_, err = RunPlugin(pluginPath, "set", filePath, key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

func DeleteMetadata(filePath, key string, opts Options) error {
	if !opts.FileOnly {
		if err := DeleteXattr(filePath, key); err != nil {
			return err
		}
	}

	if !opts.XattrOnly {
		mimeType, err := getMimeType(filePath, opts)
		if err != nil {
			return err
		}

		pluginPath, err := FindPlugin(mimeType)
		if err != nil {
			return err
		}

		_, err = RunPlugin(pluginPath, "delete", filePath, key, "")
		if err != nil {
			return err
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
