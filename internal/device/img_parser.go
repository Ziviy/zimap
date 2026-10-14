package device

import (
	"bytes"
	"io"
	"os"
	"strings"
	"unicode"
)

// ReadMapInternalName reads the internal map description embedded in a Garmin .img file header.
// In Garmin image headers (GMP/DSKIMG format), offset 0x49 (73 bytes) contains an ASCII/UTF-8
// description string (up to 30-50 characters) set during compilation (e.g. mkgmap --description).
func ReadMapInternalName(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return ParseMapInternalNameFromReader(f)
}

// ParseMapInternalNameFromReader extracts the map title from a reader of the first bytes of a .img file.
func ParseMapInternalNameFromReader(r io.Reader) (string, error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	if n < 0x49+2 {
		return "", nil
	}

	// Read slice starting from offset 0x49 (decimal 73)
	maxLen := 50
	end := 0x49 + maxLen
	if end > n {
		end = n
	}

	raw := buf[0x49:end]

	// Find null terminator if present
	if nullIdx := bytes.IndexByte(raw, 0x00); nullIdx != -1 {
		raw = raw[:nullIdx]
	}

	str := strings.TrimSpace(string(raw))

	// Check that the string is mostly printable text
	if len(str) < 2 {
		return "", nil
	}

	for _, r := range str {
		if !unicode.IsPrint(r) && r != ' ' {
			return "", nil
		}
	}

	return str, nil
}
