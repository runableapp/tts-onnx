package main

import (
	"fmt"
	"os"
	"runtime"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

func main() {
	fmt.Printf("goos=%s goarch=%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CGO_ENABLED=%s\n", os.Getenv("CGO_ENABLED"))
	fmt.Printf("LD_LIBRARY_PATH=%s\n", os.Getenv("LD_LIBRARY_PATH"))
	fmt.Printf("sherpa_onnx_version=%s git_sha=%s git_date=%s\n", sherpa.GetVersion(), sherpa.GetGitSha1(), sherpa.GetGitDate())
	fmt.Println("runtime-check: use this command to validate native sherpa/onnx library visibility before enabling engine=sherpa-onnx.")
}
