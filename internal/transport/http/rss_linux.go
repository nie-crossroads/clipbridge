package httptransport

import (
	"os"
	"strconv"
	"strings"
)

func processRSS() *uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return nil
	}
	resident, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil
	}
	bytes := resident * uint64(os.Getpagesize())
	return &bytes
}
