package gcp

import "github.com/crossplane/crossplane/cmd/crank/beta/importer"

const (
	network    = "network"
	subnetwork = "subnetwork"
)

var ServiceMapping = map[string]importer.Service{
	network:    &Network{},
	subnetwork: &Subnetwork{},
}
