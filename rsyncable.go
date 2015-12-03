package multigz

import (
	"io"

	gzip "github.com/klauspost/pgzip"
)

const cWINDOW_SIZE = 4096

type countWriter struct {
	io.Writer
	off int64
}

func (cw *countWriter) Write(data []byte) (int, error) {
	n, err := cw.Writer.Write(data)
	cw.off += int64(n)
	return n, err
}

type GzipWriterRsyncable struct {
	*gzip.Writer
	underw *countWriter
	window []byte
	idx    int
	sum    int
	blk    int64
}

// Create a new compressing writer that will generate a multi-gzip, segmenting
// the compressed stream in a way to be efficient when transferred over rsync
// with slight differences in the uncompressed stream.
//
// This function is similar to NewWriterLevel as it creates a multi-gzip file,
// but segenting happens at data-dependent offsets that make the compressed
// stream resynchronize after localized changes in the uncompressed stream. In
// other words, we use the same algorithm of "gzip --rsyncable", but for a
// multigz file.
func NewWriterLevelRsyncable(w io.Writer, level int) (Writer, error) {
	underw := &countWriter{Writer: w}
	bg, err := gzip.NewWriterLevel(underw, level)
	if err != nil {
		return nil, err
	}
	return &GzipWriterRsyncable{
		Writer: bg,
		underw: underw,
		window: make([]byte, cWINDOW_SIZE),
	}, nil
}

func (w *GzipWriterRsyncable) Write(data []byte) (int, error) {

	written := 0
Start:
	d := data
	for w.idx < cWINDOW_SIZE && len(d) > 0 {
		w.window[w.idx] = d[0]
		w.sum += int(d[0])
		w.idx++
		d = d[1:]
	}
	for len(d) > 0 {
		w.sum -= int(w.window[w.idx%cWINDOW_SIZE])
		w.window[w.idx%cWINDOW_SIZE] = d[0]
		w.sum += int(w.window[w.idx%cWINDOW_SIZE])
		w.idx++
		d = d[1:]
		if w.sum%cWINDOW_SIZE == 0 {
			d1 := len(data) - len(d)
			n, err := w.Writer.Write(data[:d1])
			written += n
			if err != nil {
				return 0, err
			}
			w.Writer.Flush()
			w.Writer.Close()
			w.Writer.Reset(w.underw)
			w.sum = 0
			w.idx = 0
			w.blk = w.underw.off
			data = data[d1:]
			goto Start
		}
	}

	n, err := w.Writer.Write(data)
	return written + n, err
}

func (w *GzipWriterRsyncable) Offset() Offset {
	return Offset{
		Block: int64(w.blk),
		Off:   int64(w.idx),
	}
}
