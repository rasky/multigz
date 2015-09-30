package multigz

import (
	"bufio"
	"compress/gzip"
	"io"
)

const DefaultBlockSize = 64 * 1024

type blockWriter struct {
	gz     *gzip.Writer
	underw *countWriter
	blkoff int64
}

func (bw *blockWriter) Write(data []byte) (n int, err error) {
	bw.gz.Reset(bw.underw)
	n, err = bw.gz.Write(data)
	if err != nil {
		return
	}
	err = bw.gz.Close()
	if err != nil {
		return
	}
	bw.blkoff = bw.underw.off
	return
}

type normalWriter struct {
	*bufio.Writer
	io.Closer
	blkw *blockWriter
}

// Create a new compressing writer that will generate a multi-gzip, segmenting
// the compressed stream at fixed offsets. This is similar to gzip.NewWriterLevel,
// but takes an additional argument that specifies the size of each gzip block.
// You can use multigz.DefaultBlockSize as a reasonable default (64kb) that
// balances decompression speed and compression overhead.
func NewWriterLevel(w io.Writer, level int, blocksize int) (Writer, error) {
	underw := &countWriter{Writer: w}
	gz, err := gzip.NewWriterLevel(underw, level)
	if err != nil {
		return nil, err
	}
	blockw := &blockWriter{
		gz:     gz,
		underw: underw,
	}
	buf := bufio.NewWriterSize(blockw, blocksize)
	return normalWriter{
		Writer: buf,
		Closer: gz,
		blkw:   blockw,
	}, nil
}

func (nw normalWriter) Offset() Offset {
	return Offset{
		Block: nw.blkw.blkoff,
		Off:   int64(nw.Writer.Buffered()),
	}
}

func (nw normalWriter) Close() error {
	err := nw.Writer.Flush()
	if err != nil {
		return err
	}
	return nw.Closer.Close()
}
