package importer

import (
	"github.com/alecthomas/kong"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// gcpCmd arguments and flags for gcp subcommand.
type gcpCmd struct {
	Flags `embed:""`

	// Provider-specific flags
	Project string            `help:"GCP project to use for GCP resources."`
	Region  string            `help:"GCP region to use for GCP resources."`
	Tags    map[string]string `help:"Tags to apply to GCP resources."`
}

func (c *gcpCmd) Help() string {
	return `
This command generates Crossplane resource manifests for the existing GCP
resources.

Examples:
  # Generate Crossplane resource manifests for the existing GCP resources.
  crossplane beta import gcp --project=... --region=... --tags=...

  # Output to a specific file instead of stdout.
  crossplane beta import gcp -o output.yaml 
`
}

// Run import for gcp resources.
func (c *gcpCmd) Run(_ *kong.Context, _ logging.Logger) error {
	// TODO
	return nil
}
