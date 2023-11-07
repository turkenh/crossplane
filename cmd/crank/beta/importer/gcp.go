package importer

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	resourceTemplate "github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/gcp/template"
	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/resource"
	"sigs.k8s.io/yaml"
	"text/template"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/gcp"
)

// ResourceTemplate is populated with imported resources.
//
//go:embed internal/gcp/template/resource.tmpl
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
	var resources []resource.Resource

	for _, r := range c.Resources {
		if s, ok := gcp.ServiceMapping[r]; ok {
			res, err := s.GetResources(context.Background(), c.Region, c.Project, c.Filter)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get resources for %q", r))
			}
			resources = append(resources, res...)
		}
	}

	t, err := template.New("resource").Parse(resourceTemplate.Resource)
	if err != nil {
		return errors.Wrap(err, "cannot parse resource template")
	}

	buf := &bytes.Buffer{}
	for _, r := range resources {
		if r.Params != nil {
			b, err := yaml.Marshal(r.Params)
			if err != nil {
				return errors.Wrapf(err, "cannot marshal params for %v", r)
			}
			r.ProcessedParams = string(b)
		}
		if err = t.Execute(buf, r); err != nil {
			return errors.Wrapf(err, "cannot execute resource template for %v", r)
		}
		buf.Write([]byte("\n---\n"))
	}

	fmt.Println(buf.String())
	return nil
}
