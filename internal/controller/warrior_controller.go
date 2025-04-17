/*
Copyright 2025.

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

package controller

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	warriorv1alpha1 "git.gmem.ca/arch/warrior-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WarriorReconciler reconciles a Warrior object
type WarriorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Definitions to manage status conditions
const (
	// typeAvailableMemcached represents the status of the Deployment reconciliation
	typeAvailableWarrior = "Available"
	VAR_DOWNLOADER       = "DOWNLOADER"
	VAR_CONCURRENT_ITEMS = "CONCURRENT_ITEMS"
)

// +kubebuilder:rbac:groups=warrior.k8s.gmem.ca,resources=warriors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=warrior.k8s.gmem.ca,resources=warriors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=warrior.k8s.gmem.ca,resources=warriors/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *WarriorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	warrior := &warriorv1alpha1.Warrior{}
	err := r.Get(ctx, req.NamespacedName, warrior)
	if err != nil {
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-warrior", warrior.Name), Namespace: warrior.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForWarrior(warrior)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Warrior")

			// The following implementation will update the status
			meta.SetStatusCondition(&warrior.Status.Conditions, metav1.Condition{Type: typeAvailableWarrior,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", warrior.Name, err)})

			if err := r.Status().Update(ctx, warrior); err != nil {
				log.Error(err, "Failed to update Warrior status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		meta.SetStatusCondition(&warrior.Status.Conditions, metav1.Condition{
			Type:   typeAvailableWarrior,
			Status: metav1.ConditionTrue, Reason: "Reconciling",
			Message: fmt.Sprintf("Deployment for custom resource (%s) created successfully", warrior.Name),
		})

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}
	targetReplicas, err := r.replicasForWarrior(warrior)
	if err != nil {
		return ctrl.Result{}, err
	}
	resources, err := r.resourcesForWarrior(warrior)
	if err != nil {
		return ctrl.Result{}, nil
	}
	newResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resources.cpuLimit,
			"memory": resources.memoryLimit,
		},
		Requests: corev1.ResourceList{
			"cpu":    resources.cpuRequests,
			"memory": resources.memoryRequests,
		},
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-warrior", warrior.Name), Namespace: warrior.Namespace}, found)

		shouldUpdate := false
		if *found.Spec.Replicas != targetReplicas {
			found.Spec.Replicas = &targetReplicas
			shouldUpdate = true
		}
		if &newResources != found.Spec.Template.Spec.Resources {
			found.Spec.Template.Spec.Resources = &newResources
			shouldUpdate = true
		}
		if &resources.cacheSize != found.Spec.Template.Spec.Volumes[0].EmptyDir.SizeLimit {
			found.Spec.Template.Spec.Volumes[0].EmptyDir.SizeLimit = &resources.cacheSize
			shouldUpdate = true
		}
		for i, v := range found.Spec.Template.Spec.Containers[0].Env {
			if v.Name == VAR_DOWNLOADER && v.Value != warrior.Spec.Downloader {
				found.Spec.Template.Spec.Containers[0].Env[i].Value = warrior.Spec.Downloader
				shouldUpdate = true
			}
			if v.Name == VAR_CONCURRENT_ITEMS && v.Value != strconv.Itoa(warrior.Spec.Scaling.Concurrency) {
				found.Spec.Template.Spec.Containers[0].Env[i].Value = strconv.Itoa(warrior.Spec.Scaling.Concurrency)
				shouldUpdate = true
			}
		}

		if shouldUpdate {
			return r.Update(ctx, found)
		}
		return nil
	})

	if err != nil {
		log.Error(err, "Failed to update Deployment")

		if err := r.Get(ctx, req.NamespacedName, warrior); err != nil {
			log.Error(err, "Failed to re-fetch Warrior")
			return ctrl.Result{}, err
		}

		// The following implementation will update the status
		meta.SetStatusCondition(&warrior.Status.Conditions, metav1.Condition{
			Type:   typeAvailableWarrior,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to update deployment (%s): (%s)", found.Name, err)})

		if err := r.Status().Update(ctx, warrior); err != nil {
			log.Error(err, "Failed to update Warrior status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&warrior.Status.Conditions, metav1.Condition{
		Type:   typeAvailableWarrior,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas updated successfully", warrior.Name, targetReplicas),
	})

	if err := r.Status().Update(ctx, warrior); err != nil {
		log.Error(err, "Failed to update Warrior status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WarriorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&warriorv1alpha1.Warrior{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// deploymentForMemcached returns a Memcached Deployment object
func (r *WarriorReconciler) deploymentForWarrior(
	warrior *warriorv1alpha1.Warrior) (*appsv1.Deployment, error) {
	ls := r.labelsForWarrior(warrior.Spec.Project)
	replicas, err := r.replicasForWarrior(warrior)
	if err != nil {
		return nil, err
	}
	// Get the Operand image
	image, err := r.imageForWarrior(warrior.Spec.Project)
	if err != nil {
		return nil, err
	}

	resources, err := r.resourcesForWarrior(warrior)
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-warrior", warrior.Name),
			Namespace: warrior.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "cache", VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium:    "Memory",
								SizeLimit: &resources.cacheSize,
							}},
						},
					},
					// Warrior ONLY supports amd64 at the moment.
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/arch",
												Operator: "In",
												Values:   []string{"amd64"},
											},
											{
												Key:      "kubernetes.io/os",
												Operator: "In",
												Values:   []string{"linux"},
											},
										},
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "warrior",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{Name: VAR_CONCURRENT_ITEMS, Value: strconv.Itoa(warrior.Spec.Scaling.Concurrency)},
							{Name: VAR_DOWNLOADER, Value: warrior.Spec.Downloader},
						},
						VolumeMounts: []corev1.VolumeMount{
							{MountPath: "/grab/data", Name: "cache"},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								"cpu":    resources.cpuLimit,
								"memory": resources.memoryLimit,
							},
							Requests: corev1.ResourceList{
								"cpu":    resources.cpuRequests,
								"memory": resources.memoryRequests,
							},
						},
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Args: []string{"--concurrent", "$(CONCURRENT_ITEMS)", "$(DOWNLOADER)"},
					}},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(warrior, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

type resources struct {
	cpuLimit       resource.Quantity
	memoryLimit    resource.Quantity
	cpuRequests    resource.Quantity
	memoryRequests resource.Quantity
	cacheSize      resource.Quantity
}

func (r *WarriorReconciler) resourcesForWarrior(warrior *warriorv1alpha1.Warrior) (resources, error) {
	cpuLimit, err := resource.ParseQuantity(cmp.Or(warrior.Spec.Resources.Limits.CPU, "0"))
	if err != nil {
		return resources{}, fmt.Errorf("failed to parse CPU limit: %w", err)
	}
	memoryLimit, err := resource.ParseQuantity(cmp.Or(warrior.Spec.Resources.Limits.Memory, "0"))
	if err != nil {
		return resources{}, fmt.Errorf("failed to parse memory limit: %w", err)
	}
	cpuRequests, err := resource.ParseQuantity(cmp.Or(warrior.Spec.Resources.Requests.CPU, "0"))
	if err != nil {
		return resources{}, fmt.Errorf("failed to parse CPU requests: %w", err)
	}
	memoryRequests, err := resource.ParseQuantity(cmp.Or(warrior.Spec.Resources.Requests.Memory, "0"))
	if err != nil {
		return resources{}, fmt.Errorf("failed to parse memory requests: %w", err)
	}
	cacheSize, err := resource.ParseQuantity(cmp.Or(warrior.Spec.Resources.CacheSize, "500Mi"))
	if err != nil {
		return resources{}, fmt.Errorf("failed to parse cache size: %w", err)
	}
	return resources{
		cpuLimit,
		memoryLimit,
		cpuRequests,
		memoryRequests,
		cacheSize,
	}, nil
}

type stats struct {
	Counts struct {
		Done int     `json:"done"`
		Todo int     `json:"todo"`
		Rcr  float64 `json:"rcr"`
		Out  int     `json:"out"`
	} `json:"counts"`
}

func (r *WarriorReconciler) replicasForWarrior(warrior *warriorv1alpha1.Warrior) (int32, error) {
	url := fmt.Sprintf("https://legacy-api.arpa.li/%s/stats.json", warrior.Spec.Project)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return 0, err
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode > 299 {
		fmt.Println("Error fetching stats, no project?", err)
		return 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var s stats
	err = json.Unmarshal(body, &s)
	if err != nil {
		return 0, err
	}
	if s.Counts.Todo > 1_000_000 {
		return int32(warrior.Spec.Scaling.Maximum), nil
	}
	if s.Counts.Todo == 0 && s.Counts.Out > 1_000_000 {
		return int32((warrior.Spec.Scaling.Minimum + warrior.Spec.Scaling.Maximum) / 2), nil
	}
	return int32(warrior.Spec.Scaling.Minimum), nil
}

func (r *WarriorReconciler) labelsForWarrior(project string) map[string]string {
	var imageTag string
	image, err := r.imageForWarrior(project)
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{
		"app.kubernetes.io/name":       "warrior-operator",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/managed-by": "WarriorController",
		"warrior.k8s.gmem.ca/project":  project,
	}
}

func (r *WarriorReconciler) imageForWarrior(project string) (string, error) {
	var imageEnvVar = "WARRIOR_IMAGE_BASE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", fmt.Errorf("unable to find %s environment variable with the image", imageEnvVar)
	}
	return fmt.Sprintf("%s/%s-grab:latest", image, project), nil
}
