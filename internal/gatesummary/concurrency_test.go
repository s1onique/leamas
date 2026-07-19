package gatesummary

import (
	"strings"
	"sync"
	"testing"
)

func TestConcurrentDecoders(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-minimal.json")
	const goroutines = 64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := Decode(strings.NewReader(string(data)))
			if !res.Success() {
				t.Errorf("concurrent decode failed: %v", res.Diagnostics)
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentDecodersMixed(t *testing.T) {
	fixtures := []string{
		"testdata/valid/v2-minimal.json",
		"testdata/valid/v2-full.json",
		"testdata/valid/v1-full.json",
		"testdata/valid/v2-clinemm-microc3.json",
		"testdata/invalid/v2-truncated.json",
		"testdata/duplicate-keys/v2-duplicate-top-level-field.json",
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		for _, fixture := range fixtures {
			wg.Add(1)
			fx := fixture
			go func() {
				defer wg.Done()
				data := readFixture(t, fx)
				_ = Decode(strings.NewReader(string(data)))
			}()
		}
	}
	wg.Wait()
}
