package multigz

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"testing"
)

func calcHash(r io.ReadSeeker, multigz bool, t *testing.T) string {
	gz, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha1.New()
	_, err = io.Copy(hash, gz)
	if err != nil {
		t.Fatal(err)
	}
	if gz.IsProbablyMultiGzip() != multigz {
		t.Error("multigz status:", multigz, gz.IsProbablyMultiGzip())
	}
	gz.Close()
	sum := hash.Sum([]byte{})
	return hex.EncodeToString(sum)
}

func TestBasicReader(t *testing.T) {
	for idx, fn := range []string{"testdata/divina.txt.gz", "testdata/divina2.txt.gz"} {
		f, err := os.Open(fn)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		sum := calcHash(f, idx == 1, t)
		if sum != "810d873f4a55619450f6e2550b8ca0f6c2bd0baf" {
			t.Error("invalid hash for decompressed stream")
		}
	}
}

func TestIndex(t *testing.T) {
	f, err := os.Open("testdata/divina2.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	type offsets struct {
		Off Offset
		Sum string
	}
	var pos []offsets

	seed := time.Now().UnixNano()
	t.Log("using seed:", seed)
	rand.Seed(seed)
	for {
		skip := rand.Int63n(10000) + 1
		_, err := io.CopyN(ioutil.Discard, gz, skip)
		if err == io.EOF {
			break
		}

		off := gz.Offset()
		hash := sha1.New()
		io.CopyN(hash, gz, 64)
		sum := hash.Sum([]byte{})

		pos = append(pos, offsets{Off: off, Sum: hex.EncodeToString(sum)})
	}

	if !gz.IsProbablyMultiGzip() {
		t.Error("file is not detected as multigzip")
	}

	perm := rand.Perm(len(pos))
	for _, idx := range perm {
		p := pos[idx]
		gz.Seek(p.Off)
		hash := sha1.New()
		io.CopyN(hash, gz, 64)
		sum := hash.Sum([]byte{})
		if hex.EncodeToString(sum) != p.Sum {
			t.Error("invalid checksum", p, hex.EncodeToString(sum))
		}
	}
}

func TestIndexMassive(t *testing.T) {
	if testing.Short() {
		t.Skip("skip massive testing")
	}
	for i := 0; i < 100; i++ {
		TestIndex(t)
	}
}
