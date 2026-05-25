// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var atomTypes = map[int]string{
	0:  "implicit", // automatic based on atom name
	1:  "text",
	13: "jpeg",
	14: "png",
	21: "uint8",
}

// NB: atoms does not include "----", this is handled separately
var atoms = atomNames(map[string]string{
	"\xa9alb": "album",
	"\xa9art": "artist",
	"\xa9ART": "artist",
	"aART":    "album_artist",
	"\xa9day": "year",
	"\xa9nam": "title",
	"\xa9gen": "genre",
	"trkn":    "track",
	"\xa9wrt": "composer",
	"\xa9too": "encoder",
	"cprt":    "copyright",
	"covr":    "picture",
	"\xa9grp": "grouping",
	"keyw":    "keyword",
	"\xa9lyr": "lyrics",
	"\xa9cmt": "comment",
	"tmpo":    "tempo",
	"cpil":    "compilation",
	"disk":    "disc",
})

var means = map[string]bool{
	"com.apple.iTunes":          true,
	"com.mixedinkey.mixedinkey": true,
	"com.serato.dj":             true,
}

// Detect PNG image if "implicit" class is used
var pngHeader = []byte{137, 80, 78, 71, 13, 10, 26, 10}

type atomNames map[string]string

func (f atomNames) Name(n string) []string {
	res := make([]string, 1)
	for k, v := range f {
		if v == n {
			res = append(res, k)
		}
	}
	return res
}

// metadataMP4 is the implementation of Metadata for MP4 tag (atom) data.
type metadataMP4 struct {
	fileType FileType
	data     map[string]interface{}
}

// ReadAtoms reads MP4 metadata atoms from the io.ReadSeeker into a Metadata, returning
// non-nil error if there was a problem.
func ReadAtoms(r io.ReadSeeker) (Metadata, error) {
	m := metadataMP4{
		data:     make(map[string]interface{}),
		fileType: UnknownFileType,
	}
	err := m.readAtoms(r)
	return m, err
}

func (m metadataMP4) readAtoms(r io.ReadSeeker) error {
	for {
		name, size, err := readAtomHeader(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch name {
		case "meta":
			// next_item_id (int32)
			_, err := readBytes(r, 4)
			if err != nil {
				return err
			}
			fallthrough

		case "moov", "udta", "ilst":
			return m.readAtoms(r)
		}

		_, ok := atoms[name]
		var data []string
		if name == "----" {
			name, data, err = readCustomAtom(r, size)
			if err != nil {
				return err
			}

			if name != "----" {
				ok = true
				size = 0 // already read data
			}
		}

		if !ok {
			_, err := r.Seek(int64(size-8), io.SeekCurrent)
			if err != nil {
				return err
			}
			continue
		}

		err = m.readAtomData(r, name, size-8, data)
		if err != nil {
			return err
		}
	}
}

func (m metadataMP4) readAtomData(r io.ReadSeeker, name string, size uint32, processedData []string) error {
	if len(processedData) > 0 {
		m.data[name] = strings.Join(processedData, ";")
		return nil
	}

	b, err := readBytes(r, uint(size))
	if err != nil {
		return err
	}

	if name == "trkn" || name == "disk" {
		// Single data sub-atom: skip "data" header (8 bytes), then
		// version+flags(4) + locale(4), then payload[3] = track/disk,
		// payload[5] = count.
		if len(b) < 22 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for track and disk numbers, got %d", 22, len(b))
		}
		t := b[16:]
		m.data[name] = int(t[3])
		m.data[name+"_count"] = int(t[5])
		return nil
	}

	// Iterate through all data sub-atoms within this atom.
	// Some tools (e.g. mutagen) store multi-valued fields like genre as
	// multiple "data" sub-atoms inside a single parent atom.
	var texts []string
	var firstPicture *Picture
	var firstUint8 *int
	offset := 0
	for offset+8 <= len(b) {
		subSize := int(binary.BigEndian.Uint32(b[offset : offset+4]))
		if subSize < 8 || offset+subSize > len(b) {
			break
		}
		subName := string(b[offset+4 : offset+8])
		offset += subSize
		if subName != "data" {
			continue
		}

		subContent := b[offset-subSize+8 : offset]
		if len(subContent) < 4 {
			continue
		}

		class := getInt(subContent[1:4])
		contentType, ok := atomTypes[class]
		if !ok {
			continue
		}

		switch contentType {
		case "text":
			if len(subContent) >= 8 {
				texts = append(texts, string(subContent[8:]))
			}
		case "uint8":
			if firstUint8 == nil && len(subContent) >= 1 {
				v := getInt(subContent[:1])
				firstUint8 = &v
			}
		case "jpeg", "png":
			if firstPicture == nil {
				firstPicture = &Picture{
					Ext:      contentType,
					MIMEType: "image/" + contentType,
					Data:     subContent[8:],
				}
			}
		case "implicit":
			if firstPicture == nil && name == "covr" && bytes.HasPrefix(subContent, pngHeader) {
				firstPicture = &Picture{
					Ext:      "png",
					MIMEType: "image/png",
					Data:     subContent,
				}
			}
		}
	}

	if len(texts) > 0 {
		m.data[name] = strings.Join(texts, "\x00")
		return nil
	}
	if firstPicture != nil {
		m.data[name] = firstPicture
		return nil
	}
	if firstUint8 != nil {
		m.data[name] = *firstUint8
		return nil
	}
	return nil
}

func readAtomHeader(r io.ReadSeeker) (name string, size uint32, err error) {
	err = binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return
	}
	name, err = readString(r, 4)
	return
}

// Generic atom.
// Should have 3 sub atoms : mean, name and data.
// We check that mean is "com.apple.iTunes" or others and we use the subname as
// the name, and move to the data atom.
// Data atom could have multiple data values, each with a header.
// If anything goes wrong, we jump at the end of the "----" atom.
func readCustomAtom(r io.ReadSeeker, size uint32) (_ string, data []string, _ error) {
	subNames := make(map[string]string)

	for size > 8 {
		subName, subSize, err := readAtomHeader(r)
		if err != nil {
			return "", nil, err
		}

		// Remove the size of the atom from the size counter
		if size >= subSize {
			size -= subSize
		} else {
			return "", nil, errors.New("--- invalid size")
		}

		b, err := readBytes(r, uint(subSize-8))
		if err != nil {
			return "", nil, err
		}

		if len(b) < 4 {
			return "", nil, fmt.Errorf("invalid encoding: expected at least %d bytes, got %d", 4, len(b))
		}
		switch subName {
		case "mean", "name":
			subNames[subName] = string(b[4:])
		case "data":
			data = append(data, string(b[4:]))
		}
	}

	// there should remain only the header size
	if size != 8 {
		err := errors.New("---- atom out of bounds")
		return "", nil, err
	}

	if !means[subNames["mean"]] || subNames["name"] == "" || len(data) == 0 {
		return "----", nil, nil
	}

	return subNames["name"], data, nil
}

func (metadataMP4) Format() Format       { return MP4 }
func (m metadataMP4) FileType() FileType { return m.fileType }

func (m metadataMP4) Raw() map[string]interface{} { return m.data }

func (m metadataMP4) getString(n []string) string {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(string)
		}
	}
	return ""
}

func (m metadataMP4) getInt(n []string) int {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(int)
		}
	}
	return 0
}

func (m metadataMP4) Title() string {
	return m.getString(atoms.Name("title"))
}

func (m metadataMP4) Artist() string {
	return m.getString(atoms.Name("artist"))
}

func (m metadataMP4) Album() string {
	return m.getString(atoms.Name("album"))
}

func (m metadataMP4) AlbumArtist() string {
	return m.getString(atoms.Name("album_artist"))
}

func (m metadataMP4) Composer() string {
	return m.getString(atoms.Name("composer"))
}

func (m metadataMP4) Genre() string {
	return strings.Join(m.Genres(), "\x00")
}

func (m metadataMP4) Genres() []string {
	s := m.getString(atoms.Name("genre"))
	parts := strings.Split(s, "\x00")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) > 0 {
		return result
	}
	return nil
}

func (m metadataMP4) Year() int {
	date := m.getString(atoms.Name("year"))
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

func (m metadataMP4) Track() (int, int) {
	x := m.getInt([]string{"trkn"})
	if n, ok := m.data["trkn_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Disc() (int, int) {
	x := m.getInt([]string{"disk"})
	if n, ok := m.data["disk_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Lyrics() string {
	t, ok := m.data["\xa9lyr"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Comment() string {
	t, ok := m.data["\xa9cmt"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Picture() *Picture {
	v, ok := m.data["covr"]
	if !ok {
		return nil
	}
	p, _ := v.(*Picture)
	return p
}
