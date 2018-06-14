/*
Package memtest provides functions for demonstrating multiple optimizations of
memory allocation in Go.
*/
package memtest

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

// DecoderFunc is any function that reads a string of space-separated integers
// and returns the ASCII character of each integer in a string.
//
// For example, "79 75" becomes "OK".
type DecoderFunc func(io.Reader) ([]byte, error)

// DecodeSimple is a naive DecoderFunc that takes a simple, but inefficient
// approach to decoding input.
//
// Input may be of any length, provided there is sufficient memory available to
// store multiple copies and the entire output.
//
// Multiple objects are allocated implicitly and explicitly.
func DecodeSimple(r io.Reader) ([]byte, error) {
	// read all input into a byte slice
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// convert the bytes to a string
	s := string(b)

	// split the string into individual tokens
	tokens := strings.Split(s, " ")

	// parse each token
	output := make([]byte, 0)
	for _, token := range tokens {
		n, err := strconv.ParseUint(token, 10, 8)
		if err != nil {
			return nil, err
		}
		output = append(output, byte(n))
	}
	return output, nil
}

var (
	inputBuf  = make([]byte, 4096)
	outputBuf = make([]byte, 4096)
)

// DecodePrealloc is a DecoderFunc that makes use of pre-allocated buffers for
// input and output. This negates the expense of allocating these buffers on
// demand, but incurs the penalty that this function is no longer safe to use
// concurrently from multiple goroutines, as each concurrent call should corrupt
// the contents of the same buffers used by the other calls.
//
// It also requires that all input is shorter than the fixed length of the input
// buffer.
func DecodePrealloc(r io.Reader) ([]byte, error) {
	// read into static input buffer
	// nb: a single read is not guaranteed to consume all available input
	n, err := r.Read(inputBuf)
	if err != nil {
		return nil, err
	}

	s := string(inputBuf[:n])
	tokens := strings.Split(s, " ")
	for i := 0; i < len(tokens); i++ {
		n, err := strconv.ParseUint(tokens[i], 10, 8)
		if err != nil {
			return nil, err
		}
		outputBuf[i] = byte(n)
	}
	return outputBuf[:len(tokens)], nil
}

// DecodeNoAlloc is a more complex DecoderFunc that avoids all memory
// allocations by parsing the input as a byte stream - without converting to
// string.
func DecodeNoAlloc(r io.Reader) ([]byte, error) {
	// process each byte of input, tracking the current byte character c, and the
	// length of the output n.
	c, n := byte(0), 0
	for {
		// fill the input buffer
		nn, err := r.Read(inputBuf)

		// check for end of input
		if err == io.EOF {
			if c != 0 {
				// capture the last character
				outputBuf[n] = c
				n++
			}

			// truncate output buffer to the computed output length n and return
			return outputBuf[:n], nil
		}

		// check for read errors
		if err != nil {
			return nil, err
		}

		// process each byte in the input buffer
		for i := 0; i < nn; i++ {
			if inputBuf[i] == ' ' {
				outputBuf[n] = c
				c = 0
				n++

				// check we don't overflow the output buffer
				if n == len(outputBuf) {
					return outputBuf, errors.New("output buffer too small")
				}
			} else {
				// byte is an integer - increment current character
				c *= 10
				c += inputBuf[i] - '0'
			}
		}
	}
}

var (
	dynamicOutputBuf = bytes.Buffer{}
)

// DecodeDynamic functions similarly to DecodeNoAlloc except that it allows
// for parsing arbitrarily long input - as long as there is sufficient memory
// available for the output.
//
// Input is read into a small, fixed size, pre-allocated buffer.
//
// Output is written to a dynamically allocated buffer which will grow as a
// function of the output length, at a rate of O(log n). That is, an
// insignificantly small number of allocations compared to the output length.
//
// The output buffer never shrinks, so subsequent calls will never incur a
// memory allocation if their input is equal to or shorter than previous calls.
// This memory remains allocated and unusuable to the rest of the program.
//
// The dynamic buffer incurs a marginal computational performance penalty.
func DecodeDynamic(r io.Reader) ([]byte, error) {
	// reset dynamically allocated buffer
	// nb: this will not free previous allocations
	dynamicOutputBuf.Reset()

	var c byte
	for {
		// nb: tune I/O performance by adjusting the size of the static input buffer
		n, err := r.Read(inputBuf)
		if err == io.EOF {
			if c != 0 {
				// output buffer will expand as needed
				dynamicOutputBuf.WriteByte(c)
			}

			// return the bytes in the output buffer by reference (no copy)
			return dynamicOutputBuf.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
		for i := 0; i < n; i++ {
			if inputBuf[i] == ' ' {
				// output buffer will expand as needed
				dynamicOutputBuf.WriteByte(c)
				c = 0
			} else {
				c *= 10
				c += inputBuf[i] - '0'
			}
		}
	}
}
