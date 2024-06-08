package util

import (
	"errors"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path/filepath"
)

func Decompress(in string, out string) error {
	file, err := os.Open(in)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to open input file: %v", err))
	}
	defer file.Close()

	outputFile := filepath.Join(out)
	outFile, err := os.Create(outputFile)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create output file: %v", err))
	}
	defer outFile.Close()

	zs := NewZstd()
	err = zs.decompress(file, outFile)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to decompress file: %v", err))
	}

	return nil
}

type Zstd struct {
	DecoderOptions []zstd.DOption
}

// Decompress reads in, decompresses it, and writes it to out.
func (zs *Zstd) decompress(in io.Reader, out io.Writer) error {
	r, err := zstd.NewReader(in, zs.DecoderOptions...)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(out, r)
	return err
}

// NewZstd returns a new, default instance ready to be customized and used.
func NewZstd() *Zstd {
	return new(Zstd)
}
