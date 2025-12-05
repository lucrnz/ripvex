package archive

// Type represents the detected archive format
type Type int

const (
	Unknown Type = iota
	Zip
	Tar
	Gzip  // likely .tar.gz
	Bzip2 // likely .tar.bz2
	Xz    // likely .tar.xz
	Zstd  // likely .tar.zstd
)

func (a Type) String() string {
	switch a {
	case Zip:
		return "zip"
	case Tar:
		return "tar"
	case Gzip:
		return "gzip"
	case Bzip2:
		return "bzip2"
	case Xz:
		return "xz"
	case Zstd:
		return "zstd"
	default:
		return "unknown"
	}
}

// ExtractOptions configures archive extraction behavior
type ExtractOptions struct {
	StripComponents int // Number of leading path components to strip
	MaxBytes        int64
}
