package resolve

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Ready struct {
	Pipeline
}

func MakeReady(runner *v1alpha1.ServiceRunner, client client.Client) *Ready {
	return &Ready{
		Pipeline: Pipeline{
			serviceRunner: runner,
			client:        client,
		},
	}
}

var _ Resolver = &Ready{}

// We shouldn't need to make a job in this state
func (*Ready) JobName() string {
	return ""
}

// Resolve implements Resolver
func (r *Ready) Resolve(ctx context.Context) (reconcile.Result, error) {
	res := ctrl.Result{Requeue: true}

	// get the status of the update job
	prevJob, err := r.FindPreviousJob(ctx)
	if err != nil {
		return res, err
	}
	if prevJob.Status.Succeeded != 1 {
		// the create job hasn't succeeded yet; did it explicitly fail?
		for _, cond := range prevJob.Status.Conditions {
			if cond.Reason == "Failed" && cond.Status == "True" {
				return ctrl.Result{}, fmt.Errorf("Failed to read service binding information, bailing")
			}
		}
		return res, fmt.Errorf("Job not yet complete, retrying")
	}

	// read the last line of its logs; we're expecting a key-value map
	// serialized into json, which we'll convert into a secret.
	log, err := r.JobLog(ctx)
	if err != nil {
		return res, err
	}
	secretData := map[string]string{}
	err = json.Unmarshal([]byte(log), &secretData)
	if err != nil {
		return res, err
	}

	// post the data as a secret
	secret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      r.serviceRunner.Name,
			Namespace: r.serviceRunner.Namespace,
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: r.serviceRunner.APIVersion,
					Kind:       r.serviceRunner.Kind,
					Name:       r.serviceRunner.Name,
					UID:        r.serviceRunner.UID,
				},
			},
		},
		StringData: secretData,
	}
	err = r.client.Create(ctx, &secret)
	if err != nil {
		return res, err
	}

	r.serviceRunner.Status.Binding = &v1alpha1.ServiceRunnerBindingRef{Name: secret.Name}

	// delete the update job; it was successful, and we don't need it anymore
	if err = r.client.Delete(ctx, prevJob); err != nil {
		return res, err
	}

	r.serviceRunner.Status.State = PIPELINE_READY
	return res, nil
}
