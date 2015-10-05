package multigz

import (
	"io"
	"io/ioutil"
)

const DefaultPeekSize = DefaultBlockSize * 2

// Returns true if the file is (statistically) a multi-gzip. It tries to read
// peeksize bytes of decompressed data, but stopping when it sees a single gzip
// termination. Returns true if it found at least a termination, false if
// it didn't (or there is any corruption in decoding the stream).
// If the stream is full exhausted before peeksize, the function returns true
// as it it is technically still a single-block multigzip.
//
// Technically, a file is a multi-gzip even if there is just one split near the
// end of it; but the use-case we're aiming at is getting performance at
// seeking, and thus we prefer to consider files with large blocks as not proper
// multi-gzips.
func IsProbablyMultiGzip(r io.ReadSeeker, peeksize int64) bool {

	// gzip multistream requires buffered I/O to stop exactly at the stream
	// boundary.
	gz, err := NewReader(r)
	if err != nil {
		return false
	}
	defer gz.Close()

	n, err := io.CopyN(ioutil.Discard, gz, peeksize)
	if err != nil {
		return false
	}
	if n < peeksize {
		return true
	}

	return gz.IsProbablyMultiGzip()
}
