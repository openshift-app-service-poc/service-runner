/*
Copyright 2022 Red Hat.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	servicecatalogiov1alpha1 "github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	"github.com/openshift-app-service-poc/service-runner/pkg/resolve"
)

// ServiceRunnerReconciler reconciles a ServiceRunner object
type ServiceRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=servicecatalog.io,resources=servicerunners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicecatalog.io,resources=servicerunners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicecatalog.io,resources=servicerunners/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets,verbs=create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceRunner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ServiceRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	runner := &v1alpha1.ServiceRunner{}
	err := r.Client.Get(ctx, req.NamespacedName, runner)
	if err != nil {
		return ctrl.Result{}, err
	}
	res, err := resolve.GetResolver(runner, r.Client).Resolve(ctx)
	if err != nil {
		l.Error(err, "Failed to resolve service runner", "runner", runner.Name, "namespace", runner.Namespace, "stage", runner.Status.State)
	} else {
		l.Info("Resolved runner", "runner", runner.Name, "namespace", runner.Namespace, "stage", runner.Status.State)
	}
	err = r.Client.Status().Update(ctx, runner)
	if err != nil {
		res.Requeue = true
	}

	return res, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servicecatalogiov1alpha1.ServiceRunner{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
