package lib

import (
	"errors"
	"os"

	"github.com/pkg/xattr"
	"golang.org/x/sys/unix"
)

const MimetypeXattr = "user.mime_type"

var ErrXattrNotSupported = errors.New("extended attributes not supported on this system")

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
		value, err := xattr.Get(path, name)
		if err != nil {
			continue
		}
		attrs[name] = []string{string(value)}
	}

	return attrs, nil
}

func GetXattrValue(path, key string) (string, error) {
	if !xattr.XATTR_SUPPORTED {
		return "", ErrXattrNotSupported
	}

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

func SetXattr(path, key, value string) error {
	if !xattr.XATTR_SUPPORTED {
		return ErrXattrNotSupported
	}

	err := xattr.Set(path, key, []byte(value))
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return ErrXattrNotSupported
		}
		return err
	}
	return nil
}

func DeleteXattr(path, key string) error {
	if !xattr.XATTR_SUPPORTED {
		return ErrXattrNotSupported
	}

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
