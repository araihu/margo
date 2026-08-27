package main

import (
	"fmt"

	margo "github.com/araihu/margo"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "schema policy|document|site",
		Short: "Print a version-matched embedded JSON Schema",
		Long: "Print the exact JSON Schema embedded in this Margo version.\n\n" +
			"Use policy for trusted host capabilities, document for Markdown frontmatter,\n" +
			"or site for a site.yaml configuration. The output is suitable for an IDE\n" +
			"or an external JSON Schema validator.",
		Example: "  mkdir -p .schemas\n" +
			"  margo schema policy > .schemas/margo-policy.schema.json\n" +
			"  margo schema document > .schemas/margo-document.schema.json\n" +
			"  margo schema site > .schemas/margo-site.schema.json",
		Args: cobra.ExactArgs(1),
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
