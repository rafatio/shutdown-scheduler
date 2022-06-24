/*
Copyright 2022.

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
	"fmt"
	"time"

	"git.topfreegames.com/rafael.oliveira/shutdown-scheduler/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ShutdownSchedulerReconciler reconciles a ShutdownScheduler object
type ShutdownSchedulerReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	BlacklistedNamespaces []string
}

const (
	hourLayout = "15:04"
)

// +kubebuilder:rbac:groups="wildlife.io",resources=shutdownschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="wildlife.io",resources=shutdownschedulers/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

func (r *ShutdownSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	for _, blackListedNamespace := range r.BlacklistedNamespaces {
		if blackListedNamespace == req.Namespace {
			log.Info(fmt.Sprintf("skipping %s/%s since it belongs to a blacklisted namespace", req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
	}

	shutdownscheduler := &v1alpha1.ShutdownScheduler{}
	err := r.Get(ctx, req.NamespacedName, shutdownscheduler)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	finalizerName := "shutdownschedulers.wildlife.io/finalizer"

	if shutdownscheduler.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(shutdownscheduler, finalizerName) {
			controllerutil.AddFinalizer(shutdownscheduler, finalizerName)
			if err := r.Update(ctx, shutdownscheduler); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(shutdownscheduler, finalizerName) {
			err := r.upScaleLogic(ctx, *shutdownscheduler, true)
			if err != nil {
				log.Error(err, err.Error())
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(shutdownscheduler, finalizerName)
			if err := r.Update(ctx, shutdownscheduler); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(shutdownscheduler, r.Client)
	if err != nil {
		log.Error(err, "failed to initialize patch helper")
		return ctrl.Result{}, err
	}

	defer func() {
		err = patchHelper.Patch(ctx, shutdownscheduler)

		if err != nil {
			log.Error(err, "Failed to patch shutdownscheduler")
		}
		log.Info(fmt.Sprintf("finished reconcile loop for %s", shutdownscheduler.ObjectMeta.GetName()))
	}()

	if !shutdownscheduler.Status.Shutdown {
		for _, timeRange := range shutdownscheduler.Spec.TimeRange {
			if timeInRange(timeRange.WeekDay, timeRange.Start, timeRange.End, time.Now()) {

				if shutdownscheduler.Spec.Resource.Kind == "Deployment" {
					object := &v1.Deployment{}
					err := r.Get(ctx, client.ObjectKey{
						Name:      shutdownscheduler.Spec.Resource.Name,
						Namespace: shutdownscheduler.Spec.Resource.Namespace,
					}, object)
					if err != nil {
						log.Error(err, err.Error())
						return ctrl.Result{}, err
					}

					shutdownscheduler.Status.PreviousReplicas = int(*object.Spec.Replicas)

					if _, ok := object.Annotations["fluxcd.io/sync-checksum"]; ok {
						object.Annotations["fluxcd.io/ignore"] = "true"
					}
					var replicas int32 = 0
					object.Spec.Replicas = &replicas

					err = r.Update(ctx, object)
					if err != nil {
						log.Error(err, err.Error())
						return ctrl.Result{}, err
					}

					shutdownscheduler.Status.Shutdown = true

				} else if shutdownscheduler.Spec.Resource.Kind == "StatefulSet" {
					object := &v1.StatefulSet{}
					err := r.Get(ctx, client.ObjectKey{
						Name:      shutdownscheduler.Spec.Resource.Name,
						Namespace: shutdownscheduler.Spec.Resource.Namespace,
					}, object)
					if err != nil {
						log.Error(err, err.Error())
						return ctrl.Result{}, err
					}

					shutdownscheduler.Status.PreviousReplicas = int(*object.Spec.Replicas)

					if _, ok := object.Annotations["fluxcd.io/sync-checksum"]; ok {
						object.Annotations["fluxcd.io/ignore"] = "true"
					}
					var replicas int32 = 0
					object.Spec.Replicas = &replicas

					err = r.Update(ctx, object)
					if err != nil {
						log.Error(err, err.Error())
						return ctrl.Result{}, err
					}

					shutdownscheduler.Status.Shutdown = true
				}

			}
		}
	} else {
		shouldUpscale := true
		for _, timeRange := range shutdownscheduler.Spec.TimeRange {
			if timeInRange(timeRange.WeekDay, timeRange.Start, timeRange.End, time.Now()) {
				shouldUpscale = false
				break
			}
		}

		err := r.upScaleLogic(ctx, *shutdownscheduler, shouldUpscale)
		if err != nil {
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ShutdownSchedulerReconciler) upScaleLogic(ctx context.Context, shutdownscheduler v1alpha1.ShutdownScheduler, shouldUpscale bool) error {
	log := log.FromContext(ctx)

	if shouldUpscale {
		if shutdownscheduler.Spec.Resource.Kind == "Deployment" {
			object := &v1.Deployment{}
			err := r.Get(ctx, client.ObjectKey{
				Name:      shutdownscheduler.Spec.Resource.Name,
				Namespace: shutdownscheduler.Spec.Resource.Namespace,
			}, object)
			if err != nil {
				log.Error(err, err.Error())
				return err
			}

			delete(object.Annotations, "fluxcd.io/ignore")

			var replicas int32 = int32(shutdownscheduler.Status.PreviousReplicas)
			object.Spec.Replicas = &replicas

			err = r.Update(ctx, object)
			if err != nil {
				log.Error(err, err.Error())
				return err
			}

			shutdownscheduler.Status.Shutdown = false

		} else if shutdownscheduler.Spec.Resource.Kind == "StatefulSet" {
			object := &v1.StatefulSet{}
			err := r.Get(ctx, client.ObjectKey{
				Name:      shutdownscheduler.Spec.Resource.Name,
				Namespace: shutdownscheduler.Spec.Resource.Namespace,
			}, object)
			if err != nil {
				log.Error(err, err.Error())
				return err
			}

			delete(object.Annotations, "fluxcd.io/ignore")

			var replicas int32 = int32(shutdownscheduler.Status.PreviousReplicas)
			object.Spec.Replicas = &replicas

			err = r.Update(ctx, object)
			if err != nil {
				log.Error(err, err.Error())
				return err
			}

			shutdownscheduler.Status.Shutdown = false
		}
	}
	return nil
}

func timeInRange(day int, start string, end string, currentTime time.Time) bool {
	startHour, _ := time.Parse(hourLayout, start)
	endHour, _ := time.Parse(hourLayout, end)
	currentHour, _ := time.Parse(hourLayout, currentTime.Format(hourLayout))

	if currentTime.Weekday() != time.Weekday(day) {
		return false
	}

	if (currentHour.After(startHour) || currentHour.Equal(startHour)) && currentHour.Before(endHour) {
		return true
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShutdownSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ShutdownScheduler{}).
		Complete(r)
}
