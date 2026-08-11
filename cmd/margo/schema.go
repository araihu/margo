package main

import (
	"fmt"

	margo "github.com/araihu/margo"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "schema policy|document",
		Short: "Print a version-matched embedded JSON Schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			data, err := margo.Schema(margo.SchemaKind(args[0]))
			if err != nil {
				return fmt.Errorf("cli.schema_invalid: %w", err)
			}
			_, err = command.OutOrStdout().Write(data)
			return err
		},
	}
	return command
}
