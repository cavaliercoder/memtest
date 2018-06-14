package memtest

import (
	"bytes"
	"fmt"
	"log"
	"sync"
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
		{"Concurrent", NewDecodeConcurrent()},
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

func TestNewDecoderConcurrent(t *testing.T) {
	goroutines := 64
	testCount := 8
	ch := make(chan []byte, 0)
	for i := 0; i < goroutines; i++ {
		go func() {
			// init new DecoderFunc with goroutine-local buffers
			f := NewDecodeConcurrent()
			r := bytes.NewReader(nil)

			// reuse DecoderFunc in this goroutine
			for i := 0; i < testCount; i++ {
				r.Reset(testInput)

				// decode
				b, err := f(r)
				if err != nil {
					t.Fatal(err)
				}

				// send output for validation
				ch <- b
			}
		}()
	}

	for i := 0; i < goroutines*testCount; i++ {
		b := <-ch
		if !bytes.Equal(b, testOutput) {
			// output will be corrupted if DecoderFunc is not concurrency-safe
			t.Errorf("expected: %s\ngot: %s", testOutput, b)
		}
	}
}

func ExampleNewDecoderConcurrent() {
	// create a WaitGroup so we can signal when each goroutine completes
	wg := &sync.WaitGroup{}
	for i := 0; i < 64; i++ {
		// spawn a new goroutine
		wg.Add(1)
		go func(i int) {
			// create a concurrency-safe DecoderFunc
			// any other func will likely result in corruption
			f := NewDecodeConcurrent()

			// parse test input
			r := bytes.NewReader(testInput)
			b, err := f(r)
			if err != nil {
				log.Fatal(err)
			}

			// print output from every 8th goroutine
			if i%8 == 0 {
				fmt.Printf("%s\n", b)
			}

			// signal we are done
			wg.Done()
		}(i)
	}

	// wait for all goroutines to finish
	wg.Wait()

	// output:
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit.
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

var (
	startGoroutines = &sync.Once{}
	goroutineCount  = 64
	inputChannel    = make(chan []byte, goroutineCount)
	outputChannel   = make(chan []byte, goroutineCount)
)

func BenchmarkDecodeConcurrent(b *testing.B) {
	// start goroutines once
	startGoroutines.Do(func() {
		for i := 0; i < goroutineCount; i++ {
			go func() {
				f := NewDecodeConcurrent()
				r := bytes.NewReader(nil)
				for testInput := range inputChannel {
					r.Reset(testInput)

					// work
					v, err := f(r)
					if err != nil {
						b.Fatal(err)
					}
					outputChannel <- v
				}
			}()
		}
	})
	b.ResetTimer()

	// send tests
	go func() {
		for i := 0; i < b.N; i++ {
			inputChannel <- testInput
		}
	}()

	// wait for responses
	for i := 0; i < b.N; i++ {
		<-outputChannel
	}
}
