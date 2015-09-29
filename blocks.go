package multigz

import (
	"bufio"
	"compress/gzip"
	"io"
)

type blockWriter struct {
	gz     *gzip.Writer
	underw io.Writer
}

const DefaultBlockSize = 64 * 1024

type closeFlusher struct {
	*bufio.Writer
	io.Closer
}

func NewWriterLevel(w io.Writer, level int, blocksize int) (io.WriteCloser, error) {
	gz, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewWriterSize(blockWriter{gz, w}, blocksize)
	return closeFlusher{Writer: buf, Closer: gz}, nil
}

func (bw blockWriter) Write(data []byte) (n int, err error) {
	bw.gz.Reset(bw.underw)
	n, err = bw.gz.Write(data)
	if err != nil {
		return
	}
	return n, bw.gz.Close()
}

func (cf closeFlusher) Close() error {
	err := cf.Writer.Flush()
	if err != nil {
		return err
	}
	return cf.Closer.Close()
}
