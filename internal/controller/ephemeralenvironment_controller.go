/*
Copyright 2026.

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
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ephemeralv1alpha1 "github.com/bugimprover/ephemeral-operator/api/v1alpha1"
)

// EphemeralEnvironmentReconciler reconciles a EphemeralEnvironment object
type EphemeralEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const ephemeralFinalizer = "ephemeral.myexample.com/finalizer"

// +kubebuilder:rbac:groups=ephemeral.myexample.com,resources=ephemeralenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ephemeral.myexample.com,resources=ephemeralenvironments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ephemeral.myexample.com,resources=ephemeralenvironments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EphemeralEnvironment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *EphemeralEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var obj ephemeralv1alpha1.EphemeralEnvironment
	var log = logf.FromContext(ctx)

	// Fetch the EphemeralEnvironment instance ; get my kube object ; handle error if not found
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		if apierrors.IsNotFound(err) { // if the object is not found, print the log and return
			log.Info("EphemeralEnvironment not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get EphemeralEnvironment") // if there is an error other than not found, print the log and return the error
		return ctrl.Result{}, err
	}

	// Get annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	const targetAnnotation = "ephemeral.myexample.com/cleanup-on-delete"
	if !obj.DeletionTimestamp.IsZero() {
		if annotations[targetAnnotation] == "true" {
			if obj.Spec.Action == "ScaleToZero" {
				if err := r.scaleResourcesToZero(ctx, obj); err != nil {
					log.Error(err, "Failed to scale resources to zero")
					return ctrl.Result{}, err
				}
			}

			if obj.Spec.Action == "Delete" {
				if err := r.deleteNamespace(ctx, obj); err != nil {
					log.Error(err, "Failed to delete namespace")
					return ctrl.Result{}, err
				}
			}
		}

		controllerutil.RemoveFinalizer(&obj, ephemeralFinalizer)
		if err := r.Update(ctx, &obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Calculate the ExpiryTime and set the time and status to the object
	if obj.Status.ExpiryTime == nil {
		createTime := obj.CreationTimestamp.Time
		ttl := obj.Spec.TTL.Duration
		resultTime := metav1.NewTime(createTime.Add(ttl))
		obj.Status.ExpiryTime = &resultTime
		// Update the status of the object
		if res, err := r.updateStatus(ctx, &obj, "Active", ""); err != nil {
			return res, err
		}
	}

	// Print the log of the object
	log.Info("Reconciling EphemeralEnvironment",
		"name", obj.Name,
		"TTL", obj.Spec.TTL,
		"targetNamespace", obj.Spec.TargetNamespace,
		"action", obj.Spec.Action)

	// Calculate the remaining time until the ExpiryTime and requeue if necessary
	remainingTime := time.Until(obj.Status.ExpiryTime.Time)

	if _, exists := annotations[targetAnnotation]; !exists {
		// Adding annotation
		annotations[targetAnnotation] = "false"
		obj.SetAnnotations(annotations)

		// Save changes
		if err := r.Update(ctx, &obj); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Checking and adding the Finalizer
	if !controllerutil.ContainsFinalizer(&obj, ephemeralFinalizer) {
		controllerutil.AddFinalizer(&obj, ephemeralFinalizer)
		if err := r.Update(ctx, &obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If the remaining time is less than or equal to zero
	if remainingTime <= 0 {

		if obj.Status.State == "Expired" {
			return ctrl.Result{}, nil
		}
		log.Info("EphemeralEnvironment has expired, performing cleanup",
			"name", obj.Name,
			"targetNamespace", obj.Spec.TargetNamespace)

		if res, err := r.updateStatus(ctx, &obj, "Expiring", "TTL expired"); err != nil {
			return res, err
		}

		// scaling resources
		if obj.Spec.Action == "ScaleToZero" {
			if err := r.scaleResourcesToZero(ctx, obj); err != nil {
				log.Error(err, "Failed to scale resources to zero")
				return r.updateStatus(ctx, &obj, "ScaleToZeroFailed", err.Error())
			}
			if res, err := r.updateStatus(ctx, &obj, "Expired", "ScaleToZero was done"); err != nil {
				return res, err
			}
		}

		// deleteing namespace
		if obj.Spec.Action == "Delete" {
			if err := r.deleteNamespace(ctx, obj); err != nil {
				log.Error(err, "Failed to delete namespace")
				return r.updateStatus(ctx, &obj, "DeleteFailed", err.Error())
			}
			if res, err := r.updateStatus(ctx, &obj, "Expired", "Namespace was deleted"); err != nil {
				return res, err
			}
		}

		return ctrl.Result{}, nil // return without requeing
	}

	log.Info("EphemeralEnvironment is still active, requeuing",
		"name", obj.Name,
		"remainingTime", remainingTime)
	return ctrl.Result{RequeueAfter: remainingTime}, nil
}

func (r *EphemeralEnvironmentReconciler) updateStatus(
	ctx context.Context,
	obj *ephemeralv1alpha1.EphemeralEnvironment,
	status string,
	reason string,
) (ctrl.Result, error) {

	var log = logf.FromContext(ctx)
	obj.Status.State = status
	obj.Status.Reason = reason
	// Update the status of the object
	if err := r.Status().Update(ctx, obj); err != nil {
		switch {
		case apierrors.IsConflict(err):
			log.Info("EphemeralEnvironment has been changed")
			return ctrl.Result{Requeue: true}, err
		case apierrors.IsNotFound(err):
			log.Info("EphemeralEnvironment not found")
			return ctrl.Result{}, nil
		default:
			log.Error(err, "Failed to update EphemeralEnvironment")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// ScaleToZero function to scale the target namespace to zero replicas
func (r *EphemeralEnvironmentReconciler) scaleResourcesToZero(ctx context.Context, obj ephemeralv1alpha1.EphemeralEnvironment) error {
	// Implement the logic to scale the target namespace to zero replicas
	var deployments appsv1.DeploymentList
	var statefulsets appsv1.StatefulSetList

	var log = logf.FromContext(ctx)

	if obj.Spec.TargetNamespace == nil {
		log.Info("TargetNamespace is not set")
		return fmt.Errorf("targetNamespace is required")
	}

	err := r.List(ctx, &statefulsets, client.InNamespace(*obj.Spec.TargetNamespace))
	if err != nil { // if error on request, print the log and return
		log.Error(err, "Something went wrong on getting statefulsets")
		return err
	}
	if len(statefulsets.Items) > 0 {
		for i := range statefulsets.Items {
			stfs := &statefulsets.Items[i]
			log.Info("Found statefulset", "name", stfs.Name)
			stfs.Spec.Replicas = ptr.To(int32(0)) // set pod amount to 0
			if err := r.Update(ctx, stfs); err != nil {
				log.Error(err, "Failed to update statefulset", "name", stfs.Name)
				return err
			}
		}
		log.Info("Successfully scaled target namespace to zero replicas",
			"targetNamespace", obj.Spec.TargetNamespace)
	}

	err = r.List(ctx, &deployments, client.InNamespace(*obj.Spec.TargetNamespace))
	if err != nil { // if error on request, print the log and return
		log.Error(err, "Something went wrong on getting deployments")
		return err
	}
	if len(deployments.Items) > 0 {
		for i := range deployments.Items {
			depl := &deployments.Items[i]
			log.Info("Found deployment", "name", depl.Name)
			depl.Spec.Replicas = ptr.To(int32(0)) // set pod amount to 0
			if err := r.Update(ctx, depl); err != nil {
				log.Error(err, "Failed to update deployment", "name", depl.Name)
				return err
			}
		}
		log.Info("Successfully scaled target namespace to zero replicas",
			"targetNamespace", obj.Spec.TargetNamespace)
	}
	return nil
}

func (r *EphemeralEnvironmentReconciler) deleteNamespace(ctx context.Context, obj ephemeralv1alpha1.EphemeralEnvironment) error {
	// Implement the logic to delete namespace
	var ns corev1.Namespace
	var log = logf.FromContext(ctx)

	if obj.Spec.TargetNamespace == nil {
		log.Info("TargetNamespace is not set")
		return fmt.Errorf("targetNamespace is required")
	}

	ns.Name = *obj.Spec.TargetNamespace
	err := r.Delete(ctx, &ns)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Something went wrong on deleteing namespace")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EphemeralEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ephemeralv1alpha1.EphemeralEnvironment{}).
		Named("ephemeralenvironment").
		Complete(r)
}
