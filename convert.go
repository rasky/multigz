package multigz

import (
	"bytes"
	"errors"
	"io"

	"github.com/klauspost/compress/gzip"
)

type ConvertMode int

const (
	ConvertNormal ConvertMode = iota
	ConvertRsyncable
)

var (
	errInvalidConvertMode = errors.New("invalid convert mode specified")
)

// Convert a whole gzip file into a multi-gzip file. mode can be used to
// select between using a normal writer, or the rsync-friendly writer.
func Convert(w io.Writer, r io.Reader, mode ConvertMode) error {

	// We want to match the same algorithm originally used, to preserve
	// the rsyncable effect. The gzip library doesn't expose this data in the
	// headers, so we parse it. We don't do additional checks here, as if the
	// header is broken, gzip.NewReader will error out just afterwards.
	var gzhead [10]byte
	if _, err := io.ReadFull(r, gzhead[:]); err != nil {
		return err
	}
	comprlevel := gzip.DefaultCompression
	if gzhead[8] == 0x2 {
		comprlevel = gzip.BestCompression
	} else if gzhead[8] == 0x4 {
		comprlevel = gzip.BestSpeed
	}

	var oz io.WriteCloser
	switch mode {
	case ConvertNormal:
		oz, _ = NewWriterLevel(w, comprlevel, DefaultBlockSize)
	case ConvertRsyncable:
		oz, _ = NewWriterLevelRsyncable(w, comprlevel)
	default:
		return errInvalidConvertMode
	}
	defer oz.Close()

	fz, err := gzip.NewReader(io.MultiReader(bytes.NewReader(gzhead[:]), r))
	if err != nil {
		return nil
	}
	defer fz.Close()

	if _, err = io.Copy(oz, fz); err != nil {
		return err
	}

	return nil
}
