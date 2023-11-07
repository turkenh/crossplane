package importer

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/gcp"
)

// ResourceTemplate is populated with imported resources.
//
//go:embed internal/gcp/templates/resource.tmpl
var ResourceTemplate string

// gcpCmd arguments and flags for gcp subcommand.
type gcpCmd struct {
	Flags `embed:""`

	// Provider-specific flags
	Project string `help:"GCP project to use for GCP resources."`
	Region  string `help:"GCP region to use for GCP resources."`
	Filter  string `help:"Filter to apply to GCP resources."`
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
	var resources []Resource

	for _, r := range c.Resources {
		if s, ok := gcp.ServiceMapping[r]; ok {
			res, err := s.GetResources(context.Background(), c.Region, c.Project, c.Filter)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get resources for %q", r))
			}
			resources = append(resources, res...)
		}
	}

	return nil
}

type Resource struct {
	APIVersion   string
	Kind         string
	ExternalName string
	Params       map[string]any
}

type Service interface {
	// GetResources returns a list of resources that can be imported.
	GetResources(ctx context.Context, region, project, filter string) ([]Resource, error)
}
