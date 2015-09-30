package multigz

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

func testWriter(t *testing.T, mode ConvertMode) {

	f, err := os.Open("testdata/divina.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(out.Name())

	var outw Writer
	if mode == ConvertNormal {
		outw, err = NewWriterLevel(out, -1, DefaultBlockSize)
	} else {
		outw, err = NewWriterLevelRsyncable(out, -1)
	}
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
		_, err := io.CopyN(outw, gz, skip)
		if err == io.EOF {
			break
		}

		off := outw.Offset()
		hash := sha1.New()
		io.CopyN(io.MultiWriter(outw, hash), gz, 64)
		sum := hash.Sum([]byte{})

		pos = append(pos, offsets{Off: off, Sum: hex.EncodeToString(sum)})
	}
	outw.Close()
	gz.Close()

	out.Seek(0, 0)
	rgz, err := NewReader(out)
	if err != nil {
		t.Fatal(err)
	}

	perm := rand.Perm(len(pos))
	for _, idx := range perm {
		p := pos[idx]
		rgz.Seek(p.Off)
		hash := sha1.New()
		io.CopyN(hash, rgz, 64)
		sum := hash.Sum([]byte{})
		if hex.EncodeToString(sum) != p.Sum {
			t.Error("invalid checksum", p, hex.EncodeToString(sum))
		}
	}
}

func TestWriterMassive(t *testing.T) {
	for i := 0; i < 10; i++ {
		testWriter(t, ConvertNormal)
		testWriter(t, ConvertRsyncable)
	}
}
