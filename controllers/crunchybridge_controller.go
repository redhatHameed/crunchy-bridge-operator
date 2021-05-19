/*
Copyright 2021.

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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crunchybridgev1 "github.com/CrunchyData/crunchy-bridge-operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// CrunchyBridgeReconciler reconciles a CrunchyBridge object
type CrunchyBridgeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crunchybridge.com,resources=crunchybridges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crunchybridge.com,resources=crunchybridges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crunchybridge.com,resources=crunchybridges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CrunchyBridge object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CrunchyBridgeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("crunchybridge", req.NamespacedName)
	log.Info("reconcile initiated", "object", req.String())

	// your logic here

	var crunchybridge crunchybridgev1.CrunchyBridge
	err := r.Get(ctx, req.NamespacedName, &crunchybridge)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("Crunchybridge resource not found, has been deleted")
			return ctrl.Result{}, nil
		}
		// error fetching resource instance, requeue and try again
		r.Log.Error(err, "error fetching Crunchybridge for reconcile")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crunchybridgev1.CrunchyBridge{}).
		Complete(r)
}
