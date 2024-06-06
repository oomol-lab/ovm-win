package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Xz struct{}

func (Xz) fileName() string { return ".xz" }

type MatchResult struct {
	ByName, ByStream bool
}

var xzHeader = []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}

func readAtMost(stream io.Reader, n int) ([]byte, error) {
	if stream == nil || n <= 0 {
		return []byte{}, nil
	}

	buf := make([]byte, n)
	nr, err := io.ReadFull(stream, buf)

	if err == nil ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return buf[:nr], nil
	}

	return nil, err
}

func (x Xz) xzmatch(filename string) (MatchResult, error) {
	var mr = MatchResult{
		ByName:   false,
		ByStream: false,
	}

	if strings.Contains(strings.ToLower(filename), x.fileName()) {
		mr.ByName = true
	}

	stream, err := os.Open(filename)
	if err != nil {
		return mr, fmt.Errorf("unable to open file: %v\n", err)
	}
	defer stream.Close()

	buf, err := readAtMost(stream, len(xzHeader))
	if err != nil {
		return mr, fmt.Errorf("unable to read file header: %v", err)
	}
	mr.ByStream = bytes.Equal(buf, xzHeader)

	return mr, nil
}

func XZChecker(filename string) bool {

	checker := Xz{}
	result, _ := checker.xzmatch(filename)

	if result.ByName && result.ByStream {
		return true
	} else {
		return false
	}
}
