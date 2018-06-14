package memtest

import (
	"bytes"
	"testing"
)

var (
	testInput = []byte(
		`76 111 114 101 109 32 105 112 115 117 109 32 100 111 108 111 114 32 115` +
			` 105 116 32 97 109 101 116 44 32 99 111 110 115 101 99 116 101 116 117` +
			` 114 32 97 100 105 112 105 115 99 105 110 103 32 101 108 105 116 46`)

	testOutput = []byte(
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit.`)
)

func TestDecoders(t *testing.T) {
	tests := []struct {
		Name    string
		Decoder DecoderFunc
	}{
		{"Simple", DecodeSimple},
		{"Prealloc", DecodePrealloc},
		{"NoAlloc", DecodeNoAlloc},
		{"Dynamic", DecodeDynamic},
	}
	r := bytes.NewReader(nil)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r.Reset(testInput)
			b, err := test.Decoder(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, testOutput) {
				t.Errorf("expected: %s\ngot: %s", testOutput, b)
			}
		})
	}
}

var sink int

func benchmarkDecoderFunc(b *testing.B, f DecoderFunc) {
	r := bytes.NewReader(nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Reset(testInput)
		out, err := f(r)
		if err != nil {
			b.Fatal(err)
		}

		// prevent compiler from optimizing away unused function call
		sink ^= len(out)
	}
}

func BenchmarkDecodeSimple(b *testing.B) {
	benchmarkDecoderFunc(b, DecodeSimple)
}

func BenchmarkDecodePrealloc(b *testing.B) {
	benchmarkDecoderFunc(b, DecodePrealloc)
}

func BenchmarkDecodeNoAlloc(b *testing.B) {
	benchmarkDecoderFunc(b, DecodeNoAlloc)
}

func BenchmarkDecodeDynamic(b *testing.B) {
	benchmarkDecoderFunc(b, DecodeDynamic)
}
