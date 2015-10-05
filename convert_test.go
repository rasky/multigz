package multigz

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestConvert(t *testing.T) {
	for _, mode := range []ConvertMode{ConvertNormal, ConvertRsyncable} {

		f, err := os.Open("testdata/divina.txt.gz")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		w, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(w.Name())
		defer w.Close()

		err = Convert(w, f, mode)
		if err != nil {
			t.Fatal(err)
		}
		w.Close()

		f2, err := os.Open(w.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer f2.Close()

		sum := calcHash(f2, true, t)
		if sum != "810d873f4a55619450f6e2550b8ca0f6c2bd0baf" {
			t.Error("invalid hash for decompressed stream")
		}
	}
}
