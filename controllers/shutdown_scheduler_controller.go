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

	"git.topfreegames.com/rafael.oliveira/shutdown-scheduler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ShutdownSchedulerReconciler reconciles a ShutdownScheduler object
type ShutdownSchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ShutdownSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	shutdownscheduler := &v1alpha1.ShutdownScheduler{}
	err := r.Client.Get(ctx, req.NamespacedName, shutdownscheduler)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !shutdownscheduler.Status.Shutdown {
		for _, timeRange := range shutdownscheduler.Spec.TimeRange {
			if inTimeSpan(timeRange.Start, timeRange.End, metav1.Now()) {
				log.Info("Deus existe")
			}
		}
	}

	// if shutdown false
	// verifica se está na hora de iniciar
	// pegar replicas e colocar no status previousreplicas
	// verificar se contem flux e desabilitar
	// replicas 0
	// Status shutdown true

	// else
	// se está na hora de voltar
	// upscale baseado na previousreplicas
	// verifica flux e se tiver paused remover
	// status shutdown false

	log.Info("Sucesso")

	return ctrl.Result{}, nil
}

func inTimeSpan(start, end, check metav1.Time) bool {
	if start.Before(&end) {
		return !check.Before(&start) && !check.After(end.Time)
	}
	if start.Equal(&end) {
		return check.Equal(&start)
	}
	return !start.After(check.Time) || !end.Before(&check)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShutdownSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ShutdownScheduler{}).
		Complete(r)
}
