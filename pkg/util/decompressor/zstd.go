package archiver

import (
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"path/filepath"
)

type zst struct{}

func newZst() *zst {
	return &zst{}
}

func (z *zst) decompress(in io.Reader, out io.Writer) error {
	readed, _ := z.openReader(in)
	defer readed.Close()

	if _, err := io.Copy(out, readed); err != nil {
		return fmt.Errorf("failed to decompress zstd stream: %w", err)
	}

	return nil
}

func (z *zst) openReader(in io.Reader) (io.ReadCloser, error) {
	zr, err := zstd.NewReader(in)
	if err != nil {
		return nil, fmt.Errorf("failed to new zstd reader stream: %w", err)
	}

	return io.NopCloser(zr), nil
}

func (z *zst) checkExt(filename string) error {
	ext := filepath.Ext(filename)
	if ext != ".zst" && ext != ".zstd" {
		return fmt.Errorf("filename %s must have a .zst extension", filename)
	}

	return nil
}
