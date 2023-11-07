package gcp

import (
	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/resource"
)

const (
	network    = "network"
	subnetwork = "subnetwork"
)

var ServiceMapping = map[string]resource.Service{
	network:    &Network{},
	subnetwork: &Subnetwork{},
}
