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

func ExtractSourceCode(targetPath string) error {
	reader, err := zip.NewReader(bytes.NewReader(sourceCodeZip), int64(len(sourceCodeZip)))
	if err != nil {
		return fmt.Errorf("open embedded zip: %w", err)
	}

	var targetFile *zip.File
	for _, f := range reader.File {
		// Only extract the file named sourcecode.vhdx
		if f.Name == "sourcecode.vhdx" {
			targetFile = f
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("sourcecode.vhdx not found in zip")
	}

	sourceCodeDiskPath := filepath.Join(targetPath, targetFile.Name)

	if err := os.MkdirAll(filepath.Dir(sourceCodeDiskPath), 0755); err != nil {
		return fmt.Errorf("create directory for vhdx: %w", err)
	}

	rc, err := targetFile.Open()
	if err != nil {
		return fmt.Errorf("open zip file entry: %w", err)
	}
	defer rc.Close()

	out, err := os.OpenFile(sourceCodeDiskPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create target file %s: %w", sourceCodeDiskPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("copy content to %s: %w", sourceCodeDiskPath, err)
	}

	return nil
}
