package archiver

import (
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

func Zstd(source, target string, overwrite bool) error {
	fd, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", source, err)
	}
	defer fd.Close()

	if overwrite {
		if err := util.Exists(target); err == nil {
			_ = os.RemoveAll(target)
		}
	}

	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", target, err)
	}
	defer out.Close()

	zr, err := zstd.NewReader(fd)
	if err != nil {
		return fmt.Errorf("failed to new zstd reader stream: %w", err)
	}
	defer zr.Close()

	if _, err := io.Copy(out, zr); err != nil {
		return fmt.Errorf("failed to decompress zstd stream: %w", err)
	}

	return nil
}
