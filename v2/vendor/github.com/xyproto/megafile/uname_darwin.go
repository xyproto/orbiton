//go:build darwin

package megafile

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xyproto/files"
)

func uname() (string, string, string, error) {
	s, err := files.Shell("uname -a")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get system information (uname): %w", err)
	}
	fields := strings.Split(s, " ")
	if len(fields) < 3 {
		return "", "", "", errors.New("got too few fields back from uname")
	}
	// Darwin cartwheel.local 25.2.0 Darwin Kernel Version 25.2.0: Tue Nov 18 21:09:45 PST 2025; root:xnu-12377.61.12~1/RELEASE_ARM64_T6030 arm64
	if strings.Contains(hostname, ".") {
		fields := strings.SplitN(hostname, ".", 2)
		hostname = fields[0]
	}
	kernelRelease := fields[2]
	machineArch := fields[len(fields)-1]
	return hostname, kernelRelease, machineArch, nil
}
