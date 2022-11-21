package resolve

import (
	"context"
	"fmt"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Create represents the pipeline stage where we need to run the create job
type Create struct {
	Pipeline
}

func MakeCreate(runner *v1alpha1.ServiceRunner, client client.Client) *Create {
	return &Create{
		Pipeline: Pipeline{
			serviceRunner: runner,
			client:        client,
		},
	}
}

var _ Resolver = &Create{}

func (c *Create) JobName() string {
	return fmt.Sprintf("%s-create", c.serviceRunner.Name)
}

func (c *Create) Resolve(ctx context.Context) (ctrl.Result, error) {
	res := ctrl.Result{}
	job := JobTemplate(c, "/create")
	err := c.client.Create(ctx, job)
	if err != nil {
		res.Requeue = true
	}

	c.serviceRunner.Status.State = PIPELINE_CREATE
	c.serviceRunner.Status.ObservedGeneration = c.ServiceRunner().Generation
	return res, err
}
