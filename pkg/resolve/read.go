package resolve

import (
	"context"
	"fmt"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Read struct {
	Pipeline
}

func MakeRead(runner *v1alpha1.ServiceRunner, client client.Client) *Read {
	return &Read{
		Pipeline: Pipeline{
			serviceRunner: runner,
			client:        client,
		},
	}
}

// JobName implements Resolver
func (r *Read) JobName() string {
	return fmt.Sprintf("%s-read", r.serviceRunner.Name)
}

// Resolve implements Resolver
func (r *Read) Resolve(ctx context.Context) (reconcile.Result, error) {
	res := ctrl.Result{Requeue: true}

	// get the status of the create job
	createJob, err := r.FindPreviousJob(ctx)
	if err != nil {
		return res, err
	}
	if createJob.Status.Succeeded != 1 {
		// the create job hasn't succeeded yet; did it explicitly fail?
		for _, cond := range createJob.Status.Conditions {
			if cond.Reason == "Failed" && cond.Status == "True" {
				return ctrl.Result{}, fmt.Errorf("Failed to create service, bailing")
			}
		}
		return res, fmt.Errorf("Job not yet complete, retrying")
	}

	// enqueue the update job
	job := JobTemplate(r, "/read")
	err = r.client.Create(ctx, job)
	if err == nil {
		res.Requeue = false
	}

	r.serviceRunner.Status.State = PIPELINE_READ
	return res, err

}

var _ Resolver = &Read{}
