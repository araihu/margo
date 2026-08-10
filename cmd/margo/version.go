package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Module    string
	Version   string
	Commit    string
	GoVersion string
	GOOS      string
	GOARCH    string
	Engines   []string
}

func ReadBuildInfo(engines []string) BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = &debug.BuildInfo{GoVersion: runtime.Version()}
	}
	return buildInfoFromGoMetadata(info, runtime.GOOS, runtime.GOARCH, engines)
}

func buildInfoFromGoMetadata(info *debug.BuildInfo, goos, goarch string, engines []string) BuildInfo {
	build := BuildInfo{
		Module:    "github.com/araihu/margo",
		Version:   "dev",
		Commit:    "unknown",
		GoVersion: runtime.Version(),
		GOOS:      goos,
		GOARCH:    goarch,
		Engines:   append([]string(nil), engines...),
	}
	if info == nil {
		return build
	}
	if info.GoVersion != "" {
		build.GoVersion = info.GoVersion
	}
	if path := strings.TrimSuffix(info.Main.Path, "/cmd/margo"); path != "" {
		build.Module = path
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		build.Version = info.Main.Version
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			build.Commit = setting.Value
			break
		}
	}
	return build
}

func newVersionCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and compiled engine capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			build := deps.Build
			engines := append([]string(nil), build.Engines...)
			sort.Strings(engines)
			engineList := strings.Join(engines, ",")
			if engineList == "" {
				engineList = "none"
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "margo %s\nmodule %s\ncommit %s\ngo %s\nplatform %s/%s\nengines %s\n",
				build.Version,
				build.Module,
				build.Commit,
				build.GoVersion,
				build.GOOS,
				build.GOARCH,
				engineList,
			)
			return err
		},
	}
}
