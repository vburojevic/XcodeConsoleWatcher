package cli

import (
	"bufio"
	"fmt"
	"os"
)

// rotation manages per-session file rotation for tail.
type rotation struct {
	pathBuilder    func(int) (string, error)
	outputFile     *os.File
	bufferedWriter *bufio.Writer
}

func newRotation(pb func(int) (string, error)) *rotation {
	return &rotation{pathBuilder: pb}
}

func (r *rotation) Open(session int) (writer *bufio.Writer, file *os.File, path string, err error) {
	if r.pathBuilder == nil {
		return nil, nil, "", nil
	}

	if r.bufferedWriter != nil {
		r.bufferedWriter.Flush()
	}
	if r.outputFile != nil {
		r.outputFile.Close()
	}

	path, err = r.pathBuilder(session)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to build path: %w", err)
	}

	r.outputFile, err = os.Create(path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create output file: %w", err)
	}
	r.bufferedWriter = bufio.NewWriter(r.outputFile)
	return r.bufferedWriter, r.outputFile, path, nil
}

func (r *rotation) Close() {
	if r.bufferedWriter != nil {
		r.bufferedWriter.Flush()
	}
	if r.outputFile != nil {
		r.outputFile.Close()
	}
}
