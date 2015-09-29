package multigz

import (
	"os"
	"testing"
)

func TestIsMultiGzip(t *testing.T) {
	f, err := os.Open("testdata/divina.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if IsProbablyMultiGzip(f, DefaultPeekSize) {
		t.Error("divina.txt.gz detected as multigz but it isn't")
	}

	f2, err := os.Open("testdata/divina2.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if !IsProbablyMultiGzip(f2, DefaultPeekSize) {
		t.Error("divina2.txt.gz not detected as multigz but it is")
	}
}
