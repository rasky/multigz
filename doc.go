// Multigz - a pure-Go package implementing efficient seeking within gzip files
//
// Abstract
//
// This library allows to create, read and write a special kind of gzip files
// called "multi-gzip" that allow for efficient seeking. Multi-gzips are fully
// compatible with existing gzip libraries and tools, so they can be treated
// as gzip files, but this library is able to also implement efficient seeking
// within thme. So, if you are manipulating a normal gzip file, you first need
// to convert it to the multi-gzip format, before being able to seek at random
// offsets, but the conversion keep it compatible with any other existing
// software manipulating gzip files.
//
//
// How to use
//
// Most usages of seeking within gzip files do not require arbitrary random
// offsets, but only seeking to specific points within the decompressed stream;
// this library assumes this use-case. Both the Reader and the Writer types
// have a Offset() method that returns a Offset function, that represents a
// "pointer" to the current position in the decompressed stream. Reader then
// also has a Seek() method that received an Offset as argument, and seeks to
// that point.
//
// Basically, we support two main scenarios:
//
//   * If your application generates the gzip file that you need to seek into,
//     then change it to use multigz.Writer and, as you reach the points where
//     you will need to seek back to, call Offset() and store the offsets into
//     a data structure (that you can even marshal to disk if you want, like an
//     index). Then, open the multi-gzip with multigz.Reader and use Seek to
//     seek at one of previosly-generated offsets.
//
//   * If your application receives an already-compressed multi-gzip, open it
//     with multigz.Reader and scans it. When you reach points that you
//     might need to seek at later, call Offset() and store the Offset.
//     Afterwards, you can call Seek() at any time on the same Reader object
//     to seek back to the saved positions. You can serialize the Offsets
//     to disk so skip the initial indexing phase for the same file.
//
//
// Command line tool
//
// This package contains a command line tool called "multigz", which can be
// installed with the following command:
//
//      $ go get github.com/rasky/multigz/cmd/multigz
//
// The tool is mostly compatible with "gzip", supporting all its main options.
// It can be used in automatic scripts to generate multi-gzip files instead of
// gzip files. For instance, to create a .tar.gz archive where you can later
// easily seek into, use:
//
//      $ tar c <directory> | multigz -c > archive.tar.gz
//
//
// Description of multi-gzip
//
// Normally, it is impossible to seek at arbitrary offsets within a gzip stream,
// without decompressing all previous bytes. The only possible workaround is
// to generate a special gzip, in which the compressor status has been flushed
// multipled times during the stream; fir instance, if we flush the status
// every 64k bytes, we will need to decompress at most 64k before getting to
// any point in the decompressed stream, putting an upper bound to required
// random seeking.
//
// Flushing a deflate stream is not really supported by the deflate format
// itself, but fortunately gzip helps here: in fact, it is possible to
// concatenate multiple gzip files, and the resulting file is a valid gzip
// file itself: a gzip-compatible decompressor is in fact expected to
// decompress multiple consecutive gzip stream until EOF is reached. This
// means that, instead of just flushing the deflate state (which would be
// incompatible with existing decompressors), we flush and close the gzip
// stream, and start a new one within the same file. The resulting file is
// a valid gzip file, compatible with all existing gzip libraries and tools,
// but it can be efficiently seeked by knowing in advance where each internal
// gzip file begins.
package multigz
