package main

import "io"
import "errors"
import "testing"

// Build some fake readers and writers for testing

type FakeReader struct {
	readBytes [][]byte
	readError *error
	quitAfter int
	timesCalled int
}

func (r *FakeReader) Read(p []byte) (n int, err error) {
	if r.timesCalled >= r.quitAfter {
		return 0, *r.readError
	} else {
		read := copy(p, r.readBytes[r.timesCalled])
		r.timesCalled++
		return read, nil
	}
}

func NewFakeReader(quitAfter int, err *error) *FakeReader {
	return &FakeReader{make([][]byte, 1024, 1024), err, quitAfter, 0}
}

type FakeWriter struct {
	writtenBytes []byte
	writtenBytePtr int
}

func (w *FakeWriter) Write(p []byte) (n int, err error) {
	written := copy(w.writtenBytes[w.writtenBytePtr:], p)
	w.writtenBytePtr += written
	return written, nil
}

func NewFakeWriter() *FakeWriter {
	return &FakeWriter{make([]byte, 1024, 1024), 0}
}

// Verify that print writes to its writer

func TestPrint(t *testing.T) {
	w := NewFakeWriter()

	print(w, "Hello, world!")

	actual := string(w.writtenBytes[0:14])
	if actual != "Hello, world!\u0000" {
		t.Errorf("`print` did not write bytes correctly: '%s'", actual)
	}
}

// Verify that println writes to its writer

func TestPrintLn(t *testing.T) {
	w := NewFakeWriter()

	println(w, "Hello, world!")

	actual := string(w.writtenBytes[0:16])
	if actual != "Hello, world!\r\n\u0000" {
		t.Errorf("`println` did not write bytes correctly: '%s'", actual)
	}
}

// Verify that the main loop starts up, and will quit if 
// the FakeReader returns any kind of error

func TestMainLoopTerminatesOnReadError(t *testing.T) {
	err := errors.New("FakeError")

	r := NewFakeReader(1, &err)
	w := NewFakeWriter()

 	mainLoop(r, w)

	actual := string(w.writtenBytes[0:25])
	if actual != "mud> Error : FakeError\r\n\u0000" {
		t.Errorf("Unexpected read: '%s'", actual)
	}
	
}

// Verify that the main loop will exit cleanly if its
// Reader returns io.EOF

func TestMainLoopTerminatesOnEof(t *testing.T) {
	r := NewFakeReader(0, &io.EOF)
	w := NewFakeWriter()

	mainLoop(r, w)

	actual := string(w.writtenBytes[0:6])
	if actual != "mud> \u0000" {
		t.Errorf("Unexpected read: '%s'", actual)
	}
	
}

// Just give an un-handled command as input, so we know
// we'll see "Huh?" as the response.

func TestMainLoopHandlesInput(t *testing.T) {
	r := NewFakeReader(1, &io.EOF)
	r.readBytes[0] = []byte("foo\r\n")
	w := NewFakeWriter()

	mainLoop(r, w)

	actual := string(w.writtenBytes[0:17])
	if actual != "mud> Huh?\r\nmud> \u0000" {
		t.Errorf("Unexpected read: '%s'", actual)
	}
}
