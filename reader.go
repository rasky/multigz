package multigz

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
)

var (
	errWrongOffset = errors.New("the offset does not appear to match the gzip layout")
)

// Offset represents a specific point in the decompressed stream where we want
// to seek at. The normal way to obtain an Offset is to call Reader.Offset() of
// Writer.Offset() at the specific point in the stream we are interested into;
// later, it is possible to call Reder.Seek() passing the Offset to efficiently
// get back to that point.
type Offset struct {
	Block int64
	Off   int64
}

type countReader struct {
	R   *bufio.Reader
	Cnt *int64
}

func (cw *countReader) Read(data []byte) (n int, err error) {
	n, err = cw.R.Read(data)
	(*cw.Cnt) += int64(n)
	return
}

func (cw *countReader) ReadByte() (ch byte, err error) {
	(*cw.Cnt) += 1
	return cw.R.ReadByte()
}

// A multigz.Reader is 100% equivalent to a gzip.Reader, but allows to seek
// within the compressed file to specific positions.
//
// The idea is to use a multi-pass approach; in the first pass, you can go
// through the file and record the positions of interest by calling Offset().
// Then, you can seek to a specific offset by calling Seek().
type Reader struct {
	gz    *gzip.Reader
	ur    io.Reader
	r     io.ReadSeeker
	cnt   int64
	noff  int64
	block int64
	delim bool
}

func NewReader(r io.ReadSeeker) (*Reader, error) {
	or := new(Reader)
	or.r = r
	gz, err := gzip.NewReader(or.createUnderlyingReader())
	if err != nil {
		return nil, err
	}
	gz.Multistream(false)
	or.gz = gz
	return or, nil
}

func (or *Reader) createUnderlyingReader() io.Reader {
	or.ur = &countReader{
		R:   bufio.NewReader(or.r),
		Cnt: &or.cnt,
	}
	return or.ur
}

func (or *Reader) Read(data []byte) (int, error) {
	if or.gz == nil {
		return 0, io.EOF
	}
	nread := 0
	for len(data) > 0 {
		n, err := or.gz.Read(data)
		if err == io.EOF {
			or.noff = 0
			or.block = or.cnt
			or.gz.Close()
			if or.gz.Reset(or.ur) == io.EOF {
				or.gz = nil
				return nread, nil
			}
			or.delim = true
			or.gz.Multistream(false)
			continue
		}
		if err != nil {
			return nread + n, err
		}
		or.noff += int64(n)
		nread += n
		data = data[n:]
	}
	return nread, nil
}

func (or *Reader) Close() error {
	if or.gz == nil {
		return nil
	}
	r := or.gz
	or.gz = nil
	return r.Close()
}

func (or *Reader) Offset() Offset {
	return Offset{Block: or.block, Off: or.noff}
}

func (or *Reader) Seek(o Offset) error {
	cur := or.Offset()
	if cur.Block == o.Block && cur.Off < o.Off {
		_, err := io.CopyN(ioutil.Discard, or, o.Off-cur.Off)
		if err != nil {
			return err
		}
		return nil
	}

	or.r.Seek(o.Block, 0)
	or.cnt = o.Block

	if or.gz == nil {
		gz, err := gzip.NewReader(or.createUnderlyingReader())
		if err != nil {
			return err
		}
		or.gz = gz
	} else {
		or.gz.Close()
		if or.gz.Reset(or.createUnderlyingReader()) == io.EOF {
			or.gz = nil
			return errWrongOffset
		}
	}

	or.gz.Multistream(false)
	or.block = o.Block
	or.noff = 0

	_, err := io.CopyN(ioutil.Discard, or, o.Off)
	if err != nil {
		return err
	}

	return nil
}

// Return true if we found at least a multi-gzip separtor while reading this
// file.
// This function does not take into account the fact that short files can
// be effectively treated as multigz even if technically they aren't. Unless
// you know that you've read enough bytes out of this file, you should use
// the global function IsProbablyMultiGzip() which is a more general solution.
func (or *Reader) IsProbablyMultiGzip() bool {
	return or.delim
}
