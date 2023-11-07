package gcp

import (
	"context"
	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/resource"

	"google.golang.org/api/compute/v1"
)

func (n *Subnetwork) GetResources(ctx context.Context, region, project, filter string) ([]resource.Resource, error) {
	var resources []resource.Resource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	subnetworksList := computeService.Subnetworks.List(project, region)
	subnetworksList.Filter(filter)

	if err := subnetworksList.Pages(ctx, func(page *compute.SubnetworkList) error {
		for _, obj := range page.Items {
			resources = append(resources, resource.Resource{
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
