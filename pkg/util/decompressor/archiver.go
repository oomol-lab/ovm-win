package archiver

import (
	"fmt"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"io"
	"os"
)

// Decompressor needs to implement the Compression method and register itself through TExt
type decompressor interface {
	decompress(in io.Reader, out io.Writer) error
}

// DecompressFile decompress a compressed file through file based IO
func DecompressFile(source, destination string, overwrite bool) error {
	iface, err := createWithExt(source)
	if err != nil {
		return err
	}

	decompressor, ok := iface.(decompressor)
	if !ok {
		return fmt.Errorf("format specified by source filename is not a recognized compression algorithm: %s", source)
	}

	return fileCompressor{
		decompressor: decompressor,
		overwrite:    overwrite,
	}.decompressFile(source, destination)
}

type extensionChecker interface {
	checkExt(name string) error
}

var extCheckers = []extensionChecker{
	&zst{},
}

func createWithExt(filename string) (decompressor, error) {
	var ifce interface{}
	for _, obj := range extCheckers {
		if err := obj.checkExt(filename); err == nil {
			ifce = obj
			break
		}
	}

	switch ifce.(type) {
	case *zst:
		return newZst(), nil
	default:
		return nil, fmt.Errorf("format unrecognized by filename: %s", filename)
	}
}

// fileCompressor used to un-archive a FILE BASE archives
type fileCompressor struct {
	decompressor
	// Whether to overwrite existing files when creating files.
	overwrite bool
}

func (fc fileCompressor) decompressFile(source, destination string) error {
	if err := util.Exists(destination); !os.IsNotExist(err) && !fc.overwrite {
		return fmt.Errorf("file already exists: %s", destination)
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	return fc.decompress(in, out)
}
