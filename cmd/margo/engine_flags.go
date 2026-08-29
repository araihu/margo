package main

import (
	"context"
	"time"

	"github.com/araihu/margo/pdf"
	pdfchromium "github.com/araihu/margo/pdf/chromium"
	"github.com/araihu/margo/pdf/engines"
	"github.com/spf13/cobra"
)

type engineFlags struct {
	Mode    string
	Path    string
	Timeout time.Duration
}

func (flags *engineFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&flags.Mode, "engine", string(engines.ModeAuto), "PDF engine: auto, chromium, or native")
	command.Flags().StringVar(&flags.Path, "engine-path", "", "explicit Chromium-family executable path")
	command.Flags().DurationVar(&flags.Timeout, "engine-timeout", 30*time.Second, "Chromium PDF export timeout")
}

func selectEngine(ctx context.Context, probe engines.Probe, flags engineFlags) (pdf.Engine, engines.Candidate, error) {
	discovery, err := engines.Discover(ctx, engines.Request{Mode: engines.Mode(flags.Mode), EnginePath: flags.Path}, probe)
	if err != nil {
		return nil, engines.Candidate{}, err
	}
	engine, candidate, err := discovery.Select()
	if err != nil {
		return nil, engines.Candidate{}, err
	}
	if candidate.Name == "chromium" && flags.Timeout > 0 {
		if _, isChromium := engine.(*pdfchromium.Engine); !isChromium {
			return engine, candidate, nil
		}
		configured, configureErr := pdfchromium.New(pdfchromium.Config{ExecutablePath: candidate.Path, Timeout: flags.Timeout})
		if configureErr != nil {
			return nil, engines.Candidate{}, configureErr
		}
		engine = configured
	}
	return engine, candidate, nil
}
