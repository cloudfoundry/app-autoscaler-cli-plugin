package useragent

import (
	"fmt"
	"runtime"
)

const ProductName = "app-autoscaler-cli-plugin"

func UserAgent(version, vcsURL string) string {
	platformInfo := fmt.Sprintf("Go/%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return fmt.Sprintf("%s/%s (%s) %s", ProductName, version, vcsURL, platformInfo)
}
