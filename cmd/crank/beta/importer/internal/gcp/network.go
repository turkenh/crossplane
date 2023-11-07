package gcp

import (
	"context"
	"log"

	"google.golang.org/api/compute/v1"

	"github.com/crossplane/crossplane/cmd/crank/beta/importer"
)

func (n *Network) GetResources(ctx context.Context, _, project, filter string) ([]importer.Resource, error) {
	var resources []importer.Resource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	networksList := computeService.Networks.List(project)
	networksList.Filter(filter)

	if err := networksList.Pages(ctx, func(page *compute.NetworkList) error {
		for _, obj := range page.Items {
			resources = append(resources, importer.Resource{
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
