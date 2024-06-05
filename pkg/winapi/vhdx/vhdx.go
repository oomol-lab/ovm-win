package vhdx

import (
	"fmt"
	"syscall"
)
import "github.com/Microsoft/go-winio/vhd"

// CreateVirtualDiskFlagSupportSparseFileAnyFs
//
// winio lacks this flag
const (
	// Ref: https://github.com/microsoft/win32metadata/blob/19ceee6047a3f083bbf573400ef8596ea66ad2d1/generation/WinSDK/RecompiledIdlHeaders/um/virtdisk.h#L382-L386
	createVirtualDiskFlagSupportSparseFileAnyFs = 0x400
	blockSizeInMb uint32 = 1
)

// CreateVHDX create vhdx
func CreateVHDX(path string, maxSizeInBytes uint64) error {
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
