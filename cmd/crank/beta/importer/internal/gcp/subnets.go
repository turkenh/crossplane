package gcp

import (
	"context"

	"google.golang.org/api/compute/v1"

	"github.com/crossplane/crossplane/cmd/crank/beta/importer"
)

func (n *Subnetwork) GetResources(ctx context.Context, region, project, filter string) ([]importer.Resource, error) {
	var resources []importer.Resource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	subnetworksList := computeService.Subnetworks.List(project, region)
	subnetworksList.Filter(filter)

	if err := subnetworksList.Pages(ctx, func(page *compute.SubnetworkList) error {
		for _, obj := range page.Items {
			resources = append(resources, importer.Resource{
				APIVersion:   "compute.gcp.upbound.io/v1beta1",
				Kind:         "Subnetwork",
				ExternalName: obj.Name,
				Params: map[string]any{
					"networkRef": map[string]string{
						"name": obj.Network,
					},
					"region": region,
				},
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return resources, nil
}

type Subnetwork struct {
}
