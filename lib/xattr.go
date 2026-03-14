package lib

import (
	"errors"
	"os"
	"strings"

	"github.com/pkg/xattr"
	"golang.org/x/sys/unix"
)

const (
	MimetypeXattr     = "user.mime_type"
	MetadataNamespace = "user.metadata."
)

var ErrXattrNotSupported = errors.New("extended attributes not supported on this system")

func xattrKeyToDisplayName(key string) (string, bool) {
	if key == MimetypeXattr {
		return "mime_type", true
	}
	if strings.HasPrefix(key, MetadataNamespace) {
		return key[len(MetadataNamespace):], true
	}
	return "", false
}

func displayNameToXattrKey(name string) string {
	if name == "mime_type" {
		return MimetypeXattr
	}
	return MetadataNamespace + name
}

func GetXattr(path string) (map[string][]string, error) {
	if !xattr.XATTR_SUPPORTED {
		return nil, ErrXattrNotSupported
	}

	attrs := make(map[string][]string)

	names, err := xattr.List(path)
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return nil, ErrXattrNotSupported
		}
		return nil, err
	}

	for _, name := range names {
		displayName, ok := xattrKeyToDisplayName(name)
		if !ok {
			continue
		}
		value, err := xattr.Get(path, name)
		if err != nil {
			continue
		}
		attrs[displayName] = []string{string(value)}
	}

	return attrs, nil
}

func GetXattrValue(path, displayName string) (string, error) {
	if !xattr.XATTR_SUPPORTED {
		return "", ErrXattrNotSupported
	}

	key := displayNameToXattrKey(displayName)
	value, err := xattr.Get(path, key)
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return "", ErrXattrNotSupported
		}
		if errors.Is(err, xattr.ENOATTR) {
			return "", nil
		}
		return "", err
	}
	return string(value), nil
}

func SetXattr(path, displayName, value string) error {
	if !xattr.XATTR_SUPPORTED {
		return ErrXattrNotSupported
	}

	key := displayNameToXattrKey(displayName)
	err := xattr.Set(path, key, []byte(value))
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return ErrXattrNotSupported
		}
		return err
	}
	return nil
}

func DeleteXattr(path, displayName string) error {
	if !xattr.XATTR_SUPPORTED {
		return ErrXattrNotSupported
	}

	key := displayNameToXattrKey(displayName)
	err := xattr.Remove(path, key)
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return ErrXattrNotSupported
		}
		if errors.Is(err, xattr.ENOATTR) {
			return nil
		}
		return err
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
