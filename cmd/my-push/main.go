package main

import (
	"github.com/oneworkspacestudio/my-push/internal/app"
	"github.com/oneworkspacestudio/my-push/internal/webui"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"os"
	"path/filepath"
	"strings"
)

var version = "0.1.0-dev"

func main() {
	dataSet := false
	for _, arg := range os.Args[1:] {
		if arg == "--data" || strings.HasPrefix(arg, "--data=") {
			dataSet = true
		}
	}
	if !dataSet {
		root, e := os.UserConfigDir()
		if e != nil {
			apphost.Report(e)
			os.Exit(1)
		}
		os.Args = append(os.Args, "--data", filepath.Join(root, "OneWorkspace", "MyPush"))
	}
	if e := apphost.Main(apphost.Config{ID: "push", Name: "My Push", Version: version, Listen: "127.0.0.1:9019", Realtime: true}, webui.Files(), func(h *apphost.Host) (apphost.Application, error) { return app.New(h) }); e != nil {
		apphost.Report(e)
		os.Exit(1)
	}
}
