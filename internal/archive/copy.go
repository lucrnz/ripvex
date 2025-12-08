package archive

import (
	"context"
	"io"
)

// copyWithContext copies up to size bytes from src to dst while periodically
// checking for context cancellation. It returns the number of bytes written
// and any error encountered.
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, size int64) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer tuned for disk I/O
	var written int64
	iterCount := 0

	for written < size {
		// Check for cancellation every 10 iterations (~320KB)
		if iterCount%10 == 0 {
			if err := ctx.Err(); err != nil {
				return written, err
			}
		}
		iterCount++

		toRead := int64(len(buf))
		if remaining := size - written; remaining < toRead {
			toRead = remaining
		}

		n, err := src.Read(buf[:toRead])
		if n > 0 {
			nw, werr := dst.Write(buf[:n])
			written += int64(nw)
			if werr != nil {
				return written, werr
			}
			if nw != n {
				return written, io.ErrShortWrite
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, err
		}
	}

	return written, nil
}
