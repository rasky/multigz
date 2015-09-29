package multigz

import (
	"compress/gzip"
	"io"
)

const cWINDOW_SIZE = 4096

type GzipWriterRsyncable struct {
	*gzip.Writer
	underw io.Writer
	window []byte
	idx    int
	sum    int
}

func NewWriterLevelRsyncable(w io.Writer, level int) (*GzipWriterRsyncable, error) {
	bg, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil, err
	}
	return &GzipWriterRsyncable{Writer: bg, underw: w, window: make([]byte, cWINDOW_SIZE)}, nil
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
			data = data[d1:]
			goto Start
		}
	}

	n, err := w.Writer.Write(data)
	return written + n, err
}
