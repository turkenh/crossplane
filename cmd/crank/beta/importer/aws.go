package importer

import (
	"github.com/alecthomas/kong"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// awsCmd arguments and flags for aws subcommand.
type awsCmd struct {
	Flags `embed:""`

	// Provider-specific flags
	Region string            `help:"AWS region to use for AWS resources."`
	Tags   map[string]string `help:"Tags to apply to AWS resources."`
}

func (c *awsCmd) Help() string {
	return `
This command generates Crossplane resource manifests for the existing AWS
resources.

Examples:
  # Generate Crossplane resource manifests for the existing AWS VPCs and Subnets
  # in us-east-1 region, with tags key1=value1 and key2=value2.
  crossplane beta import aws --resources=vpc,subnet --region us-east-1 --tags "key1=value1,key2=value2"

  # Output to a specific file instead of stdout.
  crossplane beta import aws -o output.yaml --resources=vpc,subnet --region us-east-1 --tags "key1=value1,key2=value2"
`
}

// Run import for aws resources.
func (c *awsCmd) Run(_ *kong.Context, _ logging.Logger) error {
	// TODO
	return nil
}
