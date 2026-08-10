package main

import (
	"context"

	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/pdf/engines"
	"github.com/spf13/cobra"
)

type engineFlags struct {
	Mode string
	Path string
}

func (flags *engineFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&flags.Mode, "engine", string(engines.ModeAuto), "PDF engine: auto, chromium, or native")
	command.Flags().StringVar(&flags.Path, "engine-path", "", "explicit Chromium-family executable path")
}

func selectEngine(ctx context.Context, probe engines.Probe, flags engineFlags) (pdf.Engine, engines.Candidate, error) {
	discovery, err := engines.Discover(ctx, engines.Request{Mode: engines.Mode(flags.Mode), EnginePath: flags.Path}, probe)
	if err != nil {
		return nil, engines.Candidate{}, err
	}
	return discovery.Select()
}
