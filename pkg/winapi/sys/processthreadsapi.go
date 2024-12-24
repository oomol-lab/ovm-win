package sys

import (
	"github.com/oomol-lab/ovm-win/pkg/winapi"
)

// https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-isprocessorfeaturepresent
const (
	PF_SECOND_LEVEL_ADDRESS_TRANSLATION = 20
	PF_VIRT_FIRMWARE_ENABLED            = 21
)

// IsSupportedVirtualization checks if the system supports virtualization
func IsSupportedVirtualization() (vf, slat bool) {
	vf = winapi.IsProcessorFeaturePresent(PF_VIRT_FIRMWARE_ENABLED)
	slat = winapi.IsProcessorFeaturePresent(PF_SECOND_LEVEL_ADDRESS_TRANSLATION)

	return
}
