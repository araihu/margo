package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/pdf/engines"
)

type Dependencies struct {
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	SourceReader     SourceReader
	CheckAssetReader margo.CheckAssetReader
	WorkingDirectory string
	EngineProbe      engines.Probe
	NextExecutionID  func() margo.ExecutionID
	ServeSite        serveFunc
	Build            BuildInfo
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = normalizeDependencies(deps)
	version := false
	cmd := &cobra.Command{
		Use:   "margo",
		Short: "Render Markdown as HTML, PDF, or presentation decks",
		Long: "Margo checks and projects Markdown into standalone HTML, linked sites, PDFs,\n" +
			"and a versioned presentation-deck profile. Use margo help COMMAND for the\n" +
			"current flags and examples; use margo schema for IDE-facing contracts.",
		Example: "  margo check guide.md\n" +
			"  margo html guide.md --output guide.html\n" +
			"  margo site ./docs --output-dir ./dist\n" +
			"  margo doctor",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if version {
				return writeVersion(command.OutOrStdout(), deps.Build)
			}
			return command.Help()
		},
	}
	cmd.SetIn(deps.Stdin)
	cmd.SetOut(deps.Stdout)
	cmd.SetErr(deps.Stderr)
	cmd.Flags().BoolVar(&version, "version", false, "print version and compiled engine capabilities")
	cmd.AddCommand(
		newCheckCommand(deps),
		newHTMLCommand(deps),
		newSiteCommand(deps),
		newServeCommand(deps),
		newPDFCommand(deps),
		newDeckCommand(deps),
		newDoctorCommand(deps),
		newVersionCommand(deps),
		newSchemaCommand(),
	)
	return cmd
}

func Execute(ctx context.Context, deps Dependencies) error {
	return NewRootCommand(deps).ExecuteContext(ctx)
}

func normalizeDependencies(deps Dependencies) Dependencies {
	if deps.Stdin == nil {
		deps.Stdin = strings.NewReader("")
	}
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Stderr == nil {
		deps.Stderr = io.Discard
	}
	if deps.SourceReader == nil {
		deps.SourceReader = osSourceReader{}
	}
	if deps.CheckAssetReader == nil {
		deps.CheckAssetReader = margo.FilesystemCheckAssetReader{}
	}
	if deps.WorkingDirectory == "" {
		deps.WorkingDirectory, _ = os.Getwd()
	}
	if deps.NextExecutionID == nil {
		deps.NextExecutionID = randomExecutionID
	}
	if deps.ServeSite == nil {
		deps.ServeSite = defaultServeFunc(deps)
	}
	return deps
}

func randomExecutionID() margo.ExecutionID {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return ""
	}
	return margo.ExecutionID("exec-" + hex.EncodeToString(data[:]))
}
