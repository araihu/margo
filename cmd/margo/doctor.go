package main

import (
	"encoding/json"
	"fmt"

	"github.com/araihu/margo/pdf/engines"
	"github.com/spf13/cobra"
)

type doctorReport struct {
	Build      doctorBuild         `json:"build"`
	Candidates []engines.Candidate `json:"candidates"`
}

type doctorBuild struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func newDoctorCommand(deps Dependencies) *cobra.Command {
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Report available rendering engines",
		Long: "Probe the installed PDF engine candidates without rendering a document.\n" +
			"Read available=true and the candidate source/path before running pdf or\n" +
			"deck --format pdf; Margo never downloads a browser.",
		Example: "  margo doctor\n" +
			"  mkdir -p build && margo doctor --diagnostics json > build/margo-doctor.json",
		Args: diagnosticNoArgs(&diagnostics),
		RunE: func(command *cobra.Command, _ []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			discovery, err := engines.Discover(command.Context(), engines.Request{Mode: engines.ModeAuto}, deps.EngineProbe)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			report := doctorReport{
				Build:      doctorBuild{Version: deps.Build.Version, Commit: deps.Build.Commit, GoVersion: deps.Build.GoVersion, OS: deps.Build.GOOS, Arch: deps.Build.GOARCH},
				Candidates: discovery.Candidates(),
			}
			if format == diagnosticJSON {
				return json.NewEncoder(command.OutOrStdout()).Encode(report)
			}
			if _, err := fmt.Fprintf(command.OutOrStdout(), "margo %s %s/%s\n", report.Build.Version, report.Build.OS, report.Build.Arch); err != nil {
				return err
			}
			for _, candidate := range report.Candidates {
				if _, err := fmt.Fprintf(command.OutOrStdout(), "%s source=%s compiled=%t available=%t path=%q version=%q code=%q reason=%q\n",
					candidate.Name, candidate.Source, candidate.Compiled, candidate.Available, candidate.Path, candidate.Version, candidate.Code, candidate.Reason); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "output format: text or json")
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}
