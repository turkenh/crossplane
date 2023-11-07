package gcp

import (
	"context"
	"github.com/crossplane/crossplane/cmd/crank/beta/importer/internal/resource"
	"log"

	"google.golang.org/api/compute/v1"
)

func (n *Network) GetResources(ctx context.Context, _, project, filter string) ([]resource.Resource, error) {
	var resources []resource.Resource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	networksList := computeService.Networks.List(project)
	networksList.Filter(filter)

	if err := networksList.Pages(ctx, func(page *compute.NetworkList) error {
		for _, obj := range page.Items {
			resources = append(resources, resource.Resource{
				APIVersion:   "compute.gcp.upbound.io/v1beta1",
				Kind:         "Network",
				ExternalName: obj.Name,
			})
		}
		return nil
	}); err != nil {
		log.Println(err)
	}

	return resources, nil
}

type Network struct {
}
