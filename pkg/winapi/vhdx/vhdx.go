// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package vhdx

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)
import "github.com/Microsoft/go-winio/vhd"

// CreateVirtualDiskFlagSupportSparseFileAnyFs
//
// winio lacks this flag
const (
	// Ref: https://github.com/microsoft/win32metadata/blob/19ceee6047a3f083bbf573400ef8596ea66ad2d1/generation/WinSDK/RecompiledIdlHeaders/um/virtdisk.h#L382-L386
	createVirtualDiskFlagSupportSparseFileAnyFs        = 0x400
	blockSizeInMb                               uint32 = 1
)

// Create vhdx
func Create(path string, maxSizeInBytes uint64) error {
	params := vhd.CreateVirtualDiskParameters{
		Version: 2,
		Version2: vhd.CreateVersion2{
			MaximumSize:      maxSizeInBytes,
			BlockSizeInBytes: blockSizeInMb * 1024 * 1024,
		},
	}

	// Use `CreateVirtualDiskFlagSparseFile|createVirtualDiskFlagSupportSparseFileAnyFs` to create a sparse file,
	// support dynamic size (automatic shrinking)
	handle, err := vhd.CreateVirtualDisk(path, vhd.VirtualDiskAccessNone, vhd.CreateVirtualDiskFlagSparseFile|createVirtualDiskFlagSupportSparseFileAnyFs, &params)
	if err != nil {
		return fmt.Errorf("failed to create virtual disk: %w", err)
	}
	return syscall.CloseHandle(handle)
}

//go:embed sourcecode.vhdx.zip
var sourceCodeZip []byte

func ExtractSourceCode(path string) error {
	reader, err := zip.NewReader(bytes.NewReader(sourceCodeZip), int64(len(sourceCodeZip)))
	if err != nil {
		return fmt.Errorf("open embedded zip: %w", err)
	}

	for _, f := range reader.File {
		if err := extractFile(f, path); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}
	return nil
}

func extractFile(f *zip.File, destDir string) error {
	fpath := filepath.Join(destDir, f.Name)

	rel, err := filepath.Rel(destDir, fpath)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return fmt.Errorf("illegal file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(fpath, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
