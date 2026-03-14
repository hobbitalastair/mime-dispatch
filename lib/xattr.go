package lib

import (
	"encoding/csv"
	"errors"
	"strings"

	"github.com/pkg/xattr"
	"golang.org/x/sys/unix"
)

const (
	MimetypeXattr     = "user.mime_type"
	MetadataNamespace = "user.metadata."
)

var ErrXattrNotSupported = errors.New("extended attributes not supported on this system")

// encodeCSV encodes a slice of strings as a single CSV value
func encodeCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}

	// Use CSV writer to handle escaping
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Write(values)
	writer.Flush()

	// Remove trailing newline
	result := strings.TrimSuffix(buf.String(), "\n")
	return result
}

// decodeCSV decodes a CSV string into a slice of strings
func decodeCSV(value string) []string {
	if value == "" {
		return []string{}
	}

	reader := csv.NewReader(strings.NewReader(value))
	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		// If CSV parsing fails, treat as single value
		return []string{value}
	}

	if len(records[0]) == 0 {
		return []string{}
	}

	return records[0]
}

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
		// Decode CSV values
		attrs[displayName] = decodeCSV(string(value))
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

func DeleteXattr(path, displayName, value string) error {
	if !xattr.XATTR_SUPPORTED {
		return ErrXattrNotSupported
	}

	key := displayNameToXattrKey(displayName)

	// Read current value
	currentValue, err := xattr.Get(path, key)
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return ErrXattrNotSupported
		}
		if errors.Is(err, xattr.ENOATTR) {
			return nil
		}
		return err
	}

	// Decode CSV values
	values := decodeCSV(string(currentValue))

	// Find and remove the value
	newValues := []string{}
	found := false
	for _, v := range values {
		if v != value {
			newValues = append(newValues, v)
		} else {
			found = true
		}
	}

	// If value wasn't found, still return success (idempotent)
	if !found {
		return nil
	}

	// If no values remain, remove the entire xattr
	if len(newValues) == 0 {
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

	// Encode and write back updated values
	encoded := encodeCSV(newValues)
	err = xattr.Set(path, key, []byte(encoded))
	if err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return ErrXattrNotSupported
		}
		return err
	}

	return nil
}
