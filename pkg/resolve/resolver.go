package resolve

import (
	"context"
	"fmt"

	"github.com/openshift-app-service-poc/service-runner/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resolver interface {
	JobName() string
	Resolve(ctx context.Context) (ctrl.Result, error)
	ServiceRunner() *v1alpha1.ServiceRunner
}

type Pipeline struct {
	serviceRunner *v1alpha1.ServiceRunner
	client        client.Client
}

const (
	PIPELINE_CREATE = "Creating"
	PIPELINE_UPDATE = "Updating"
	PIPELINE_READ   = "Reading"
	PIPELINE_READY  = "Ready"
)

// GetResolver fetches the resolver for the current state of the service
// runner.  State transitions are defined as follows:
// - No previous state -> Create
// - Create            -> Read
// - Read              -> Ready
// - Ready             -> Update (service runner changed, we need to re-run)
// - Update            -> Read
func GetResolver(runner *v1alpha1.ServiceRunner, client client.Client) Resolver {
	switch runner.Status.State {
	case PIPELINE_CREATE:
		return MakeRead(runner, client)
	case PIPELINE_UPDATE:
		return MakeRead(runner, client)
	case PIPELINE_READ:
		return MakeReady(runner, client)
	case PIPELINE_READY:
		return MakeUpdate(runner, client)
	default:
		return MakeCreate(runner, client)
	}
}

func (p *Pipeline) FindPreviousJob(ctx context.Context) (*batchv1.Job, error) {
	job := &batchv1.Job{}
	var creator Resolver
	switch p.serviceRunner.Status.State {
	case PIPELINE_READ:
		creator = MakeRead(p.serviceRunner, p.client)
	case PIPELINE_CREATE:
		creator = MakeCreate(p.serviceRunner, p.client)
	case PIPELINE_READY:
		creator = MakeReady(p.serviceRunner, p.client)
	case PIPELINE_UPDATE:
		creator = MakeUpdate(p.serviceRunner, p.client)
	default:
		return nil, fmt.Errorf("Unexpected job state %v", p.serviceRunner.Status.State)
	}
	jobName := creator.JobName()
	namespacedname := types.NamespacedName{Namespace: p.serviceRunner.Namespace, Name: jobName}
	if err := p.client.Get(ctx, namespacedname, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (p *Pipeline) ServiceRunner() *v1alpha1.ServiceRunner {
	return p.serviceRunner
}

func (p *Pipeline) JobLog(ctx context.Context) ([]byte, error) {
	namespace := p.serviceRunner.Namespace
	job, err := p.FindPreviousJob(ctx)
	if err != nil {
		return nil, err
	}

	podList := corev1.PodList{}
	err = p.client.List(ctx, &podList)
	if err != nil {
		return nil, err
	}

	var pod *v1.Pod = nil
	for _, item := range podList.Items {
		for _, owner := range item.GetOwnerReferences() {
			if owner.Name == job.Name &&
				owner.APIVersion == job.APIVersion &&
				owner.Kind == job.Kind &&
				owner.UID == job.UID {
				pod = &item
				break
			}
		}
		if pod != nil {
			break
		}
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	one := int64(1)
	logRequest := clientSet.CoreV1().
		Pods(namespace).
		GetLogs(pod.Name, &v1.PodLogOptions{
			TailLines: &one,
		})

	return logRequest.DoRaw(ctx)
}

const CONTROL_PLANE_SECRET = "control-plane"

func JobTemplate(c Resolver, command ...string) *batchv1.Job {
	job := &batchv1.Job{}
	serviceRunner := c.ServiceRunner()
	job.Name = c.JobName()
	job.Namespace = serviceRunner.Namespace
	job.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: serviceRunner.APIVersion,
			Kind:       serviceRunner.Kind,
			Name:       serviceRunner.Name,
			UID:        serviceRunner.GetUID(),
		},
	}
	job.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:    "runner",
			Image:   serviceRunner.Spec.ServiceImage.CrudImage,
			Env:     envVars(serviceRunner.Spec.ServiceParam),
			Command: command,
		},
	}
	if len(serviceRunner.Spec.ControlPlaneSecret) != 0 {
		job.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: CONTROL_PLANE_SECRET,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: serviceRunner.Spec.ControlPlaneSecret,
					},
				},
			},
		}
		job.Spec.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{
				MountPath: "/",
				Name:      CONTROL_PLANE_SECRET,
			},
		}
	}
	job.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	return job
}

func envVars(vars map[string]string) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	for key, value := range vars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	return envVars
}
