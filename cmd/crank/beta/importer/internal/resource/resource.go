package resource

import "context"

type Resource struct {
	APIVersion      string
	Kind            string
	ExternalName    string
	Params          map[string]any
	ProcessedParams string
}

type Service interface {
	// GetResources returns a list of resources that can be imported.
	GetResources(ctx context.Context, region, project, filter string) ([]Resource, error)
}
