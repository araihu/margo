package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	WorkingDirectory string
	EngineProbe      engines.Probe
	NextExecutionID  func() margo.ExecutionID
	Build            BuildInfo
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = normalizeDependencies(deps)
	cmd := &cobra.Command{
		Use:           "margo",
		Short:         "Render Markdown as HTML, PDF, or presentation decks",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetIn(deps.Stdin)
	cmd.SetOut(deps.Stdout)
	cmd.SetErr(deps.Stderr)
	cmd.AddCommand(
		newHTMLCommand(deps),
		newPDFCommand(deps),
		newPlaceholderCommand("deck", "Render an HTML or PDF presentation deck"),
		newDoctorCommand(deps),
		newVersionCommand(deps),
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
	if deps.WorkingDirectory == "" {
		deps.WorkingDirectory, _ = os.Getwd()
	}
	if deps.NextExecutionID == nil {
		deps.NextExecutionID = randomExecutionID
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

func newPlaceholderCommand(name, description string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: description,
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("margo.command_not_implemented: %s", name)
		},
	}
}
