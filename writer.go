package multigz

import "io"

// This interface represents an object that generates a multi-gzip file.
// In addition of implementing a standard WriteCloser, it also gives access
// to a Offset method for fetching a pointer to the current position in the
// stream.
//
// In the current version, there are two different implementations of Writer:
//
//  - A writer that segments the multi-gzip file based on a fixed block
//    size. Create it with NewWriterLevel().
//  - A writer that segments the multi-gzip file making it more friendly
//    to rsync and binary-diffs. Create it wtih NewRsyncableWriter().
//
type Writer interface {
	io.WriteCloser

	// Returns an offset that points to the current point within the
	// decompressed stream.
	Offset() Offset
}
