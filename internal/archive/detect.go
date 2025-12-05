package archive

import (
	"io"
	"os"
)

// Detect reads the magic bytes from a file to determine its archive type
func Detect(path string) (Type, error) {
	f, err := os.Open(path)
	if err != nil {
		return Unknown, err
	}
	defer f.Close()

	// Read enough bytes to detect all formats (need 262 for tar ustar check).
	// Use ReadFull to avoid short reads misclassifying valid archives.
	buf := make([]byte, 262)
	n, err := io.ReadFull(f, buf)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			return Unknown, err
		}
	}
	buf = buf[:n]

	// Check ZIP: PK\x03\x04
	if len(buf) >= 4 && buf[0] == 0x50 && buf[1] == 0x4B && buf[2] == 0x03 && buf[3] == 0x04 {
		return Zip, nil
	}

	// Check GZIP: \x1f\x8b
	if len(buf) >= 2 && buf[0] == 0x1F && buf[1] == 0x8B {
		return Gzip, nil
	}

	// Check BZIP2: BZh
	if len(buf) >= 3 && buf[0] == 0x42 && buf[1] == 0x5A && buf[2] == 0x68 {
		return Bzip2, nil
	}

	// Check XZ: \xFD7zXZ\x00
	if len(buf) >= 6 && buf[0] == 0xFD && buf[1] == 0x37 && buf[2] == 0x7A &&
		buf[3] == 0x58 && buf[4] == 0x5A && buf[5] == 0x00 {
		return Xz, nil
	}

	// Check ZSTD: \x28\xB5\x2F\xFD
	if len(buf) >= 4 && buf[0] == 0x28 && buf[1] == 0xB5 && buf[2] == 0x2F && buf[3] == 0xFD {
		return Zstd, nil
	}

	// Check TAR: ustar at offset 257
	if len(buf) >= 262 {
		ustar := string(buf[257:262])
		if ustar == "ustar" {
			return Tar, nil
		}
	}

	return Unknown, nil
}
