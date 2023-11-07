package importer

import (
	"github.com/alecthomas/kong"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// azureCmd arguments and flags for azure subcommand.
type azureCmd struct {
	Flags `embed:""`

	// Provider-specific flags
	// TODO: check these make sense for azure
	Region string            `help:"Azure region to use for Azure resources."`
	Tags   map[string]string `help:"Tags to apply to Azure resources."`
}

func (c *azureCmd) Help() string {
	return `
This command generates Crossplane resource manifests for the existing Azure
resources.

Examples:
  # Generate Crossplane resource manifests for the existing Azure
  crossplane beta import azure ... TODO ...

  # Output to a specific file instead of stdout.
  crossplane beta import aws -o output.yaml ... TODO ...
`
}

// Run import for azure resources.
func (c *azureCmd) Run(_ *kong.Context, _ logging.Logger) error {
	// TODO
	return nil
}
