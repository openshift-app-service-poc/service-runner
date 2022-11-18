package resolve

import (
	"context"
	"fmt"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Update represents the pipeline stage where we need to run the update job
type Update struct {
	Pipeline
}

var _ Resolver = &Update{}

func MakeUpdate(runner *v1alpha1.ServiceRunner, client client.Client) *Update {
	return &Update{
		Pipeline: Pipeline{
			serviceRunner: runner,
			client:        client,
		},
	}
}

func (u *Update) JobName() string {
	return fmt.Sprintf("%s-update", u.serviceRunner.Name)
}

func (u *Update) Resolve(ctx context.Context) (ctrl.Result, error) {
	if u.serviceRunner.Status.ObservedGeneration == u.serviceRunner.Generation {
		// nothing to do, return ready
		return ctrl.Result{}, nil
	}
	res := ctrl.Result{Requeue: true}

	// get the status of the create job
	createJob, err := u.FindPreviousJob(ctx)
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
	job := JobTemplate(u, "/update")
	err = u.client.Create(ctx, job)
	if err == nil {
		res.Requeue = false
		u.serviceRunner.Status.State = PIPELINE_UPDATE
	}

	return res, err
}
