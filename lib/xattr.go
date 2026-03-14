package lib

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

const MimetypeXattr = "user.mime_type"

var ErrXattrNotSupported = errors.New("extended attributes not supported on this system")

func GetXattr(path string) (map[string][]string, error) {
	attrs := make(map[string][]string)

	size, err := unix.Listxattr(path, nil)
	if err != nil {
		if err == unix.ENOTSUP {
			return nil, ErrXattrNotSupported
		}
		return nil, err
	}

	if size == 0 {
		return attrs, nil
	}

	buf := make([]byte, size)
	size, err = unix.Listxattr(path, buf)
	if err != nil {
		if err == unix.ENOTSUP {
			return nil, ErrXattrNotSupported
		}
		return nil, err
	}

	names := nullSplit(string(buf[:size]))
	for _, name := range names {
		dataSize, err := unix.Getxattr(path, name, nil)
		if err != nil {
			continue
		}
		if dataSize == 0 {
			attrs[name] = []string{}
			continue
		}
		data := make([]byte, dataSize)
		_, err = unix.Getxattr(path, name, data)
		if err != nil {
			continue
		}
		attrs[name] = []string{string(data)}
	}

	return attrs, nil
}

func nullSplit(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func GetXattrValue(path, key string) (string, error) {
	size, err := unix.Getxattr(path, key, nil)
	if err != nil {
		if err == unix.ENOTSUP {
			return "", ErrXattrNotSupported
		}
		if err == unix.EINVAL {
			return "", nil
		}
		return "", err
	}
	if size == 0 {
		return "", nil
	}
	data := make([]byte, size)
	_, err = unix.Getxattr(path, key, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SetXattr(path, key, value string) error {
	err := unix.Setxattr(path, key, []byte(value), 0)
	if err != nil {
		if err == unix.ENOTSUP {
			return ErrXattrNotSupported
		}
		return err
	}
	return nil
}

func DeleteXattr(path, key string) error {
	err := unix.Removexattr(path, key)
	if err != nil {
		if err == unix.ENOTSUP {
			return ErrXattrNotSupported
		}
		if err == unix.EINVAL {
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
