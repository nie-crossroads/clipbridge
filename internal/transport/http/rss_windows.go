package httptransport

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var getProcessMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

func processRSS() *uint64 {
	// PROCESS_MEMORY_COUNTERS uses SIZE_T for its eight memory counters.
	var counters struct {
		size, pageFaultCount                                   uint32
		peakWorkingSet, workingSet, peakPagedPool, pagedPool   uintptr
		peakNonPagedPool, nonPagedPool, pagefile, peakPagefile uintptr
	}
	counters.size = uint32(unsafe.Sizeof(counters))
	if getProcessMemoryInfo.Find() != nil {
		return nil
	}
	ok, _, _ := getProcessMemoryInfo.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.size))
	if ok == 0 {
		return nil
	}
	bytes := uint64(counters.workingSet)
	return &bytes
}
