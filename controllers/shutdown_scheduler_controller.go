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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func (r *ShutdownSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	for _, blackListedNamespace := range r.BlacklistedNamespaces {
		if blackListedNamespace == req.Namespace {
			log.Info(fmt.Sprintf("skipping %s/%s since it belongs to a blacklisted namespace", req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
	}

	shutdownscheduler := &v1alpha1.ShutdownScheduler{}
	err := r.Client.Get(ctx, req.NamespacedName, shutdownscheduler)
	if err != nil {
		log.Error(err, err.Error())
		return reconcile.Result{}, err
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
					err := r.Client.Get(ctx, client.ObjectKey{
						Name:      shutdownscheduler.Spec.Resource.Name,
						Namespace: shutdownscheduler.Spec.Resource.Namespace,
					}, object)
					if err != nil {
						log.Error(err, err.Error())
						return reconcile.Result{}, err
					}

					shutdownscheduler.Status.PreviousReplicas = int(*object.Spec.Replicas)

					if _, ok := object.Annotations["fluxcd.io/sync-checksum"]; ok {
						object.Annotations["fluxcd.io/ignore"] = "true"
					}
					var replicas int32 = 0
					object.Spec.Replicas = &replicas

					err = r.Client.Update(ctx, object)
					if err != nil {
						log.Error(err, err.Error())
						return reconcile.Result{}, err
					}

					shutdownscheduler.Status.Shutdown = true

				} else if shutdownscheduler.Spec.Resource.Kind == "StatefulSet" {
					object := &v1.StatefulSet{}
					err := r.Client.Get(ctx, client.ObjectKey{
						Name:      shutdownscheduler.Spec.Resource.Name,
						Namespace: shutdownscheduler.Spec.Resource.Namespace,
					}, object)
					if err != nil {
						log.Error(err, err.Error())
						return reconcile.Result{}, err
					}

					shutdownscheduler.Status.PreviousReplicas = int(*object.Spec.Replicas)

					if _, ok := object.Annotations["fluxcd.io/sync-checksum"]; ok {
						object.Annotations["fluxcd.io/ignore"] = "true"
					}
					var replicas int32 = 0
					object.Spec.Replicas = &replicas

					err = r.Client.Update(ctx, object)
					if err != nil {
						log.Error(err, err.Error())
						return reconcile.Result{}, err
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

		if shouldUpscale {
			if shutdownscheduler.Spec.Resource.Kind == "Deployment" {
				object := &v1.Deployment{}
				err := r.Client.Get(ctx, client.ObjectKey{
					Name:      shutdownscheduler.Spec.Resource.Name,
					Namespace: shutdownscheduler.Spec.Resource.Namespace,
				}, object)
				if err != nil {
					log.Error(err, err.Error())
					return reconcile.Result{}, err
				}

				delete(object.Annotations, "fluxcd.io/ignore")

				var replicas int32 = int32(shutdownscheduler.Status.PreviousReplicas)
				object.Spec.Replicas = &replicas

				err = r.Client.Update(ctx, object)
				if err != nil {
					log.Error(err, err.Error())
					return reconcile.Result{}, err
				}

				shutdownscheduler.Status.Shutdown = false

			} else if shutdownscheduler.Spec.Resource.Kind == "StatefulSet" {
				object := &v1.StatefulSet{}
				err := r.Client.Get(ctx, client.ObjectKey{
					Name:      shutdownscheduler.Spec.Resource.Name,
					Namespace: shutdownscheduler.Spec.Resource.Namespace,
				}, object)
				if err != nil {
					log.Error(err, err.Error())
					return reconcile.Result{}, err
				}

				delete(object.Annotations, "fluxcd.io/ignore")

				var replicas int32 = int32(shutdownscheduler.Status.PreviousReplicas)
				object.Spec.Replicas = &replicas

				err = r.Client.Update(ctx, object)
				if err != nil {
					log.Error(err, err.Error())
					return reconcile.Result{}, err
				}

				shutdownscheduler.Status.Shutdown = false
			}
		}
	}

	return ctrl.Result{}, nil
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
