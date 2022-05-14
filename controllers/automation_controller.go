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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	demov1alpha1 "demo/api/v1alpha1"
)

// AutomationReconciler reconciles a Automation object
type AutomationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=demo.my.domain,resources=automations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.my.domain,resources=automations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.my.domain,resources=automations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Automation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *AutomationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("automation", req.NamespacedName)
	log.Info("Initiating automation reconciler")

	auto := &demov1alpha1.Automation{}
	err := r.Get(context.TODO(), req.NamespacedName, auto)
	if err != nil {
		if errors.IsNotFound(err) {
			//request object not found could have been deleted or modified
			return ctrl.Result{}, nil
		}
		//Error reading the object
		return ctrl.Result{}, nil
	}

	var result *ctrl.Result
	var request ctrl.Request

	// == MySQL ==========
	result, err = r.ensureSecret(request, auto, r.mysqlAuthSecret(auto))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(request, auto, r.mysqlDeployment(auto))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, auto, r.mysqlService(auto))
	if result != nil {
		return *result, err
	}

	mysqlRunning := r.isMysqlUp(auto)

	if !mysqlRunning {
		// If MySQL isn't running yet, requeue the reconcile
		// to run again after a delay
		delay := time.Second * time.Duration(5)

		log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// == autoisitors Backend  ==========
	result, err = r.ensureDeployment(request, auto, r.backendDeployment(auto))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, auto, r.backendService(auto))
	if result != nil {
		return *result, err
	}

	err = r.updateBackendStatus(auto)
	if err != nil {
		// Requeue the request if the status could not be updated
		return ctrl.Result{}, err
	}

	result, err = r.handleBackendChanges(auto)
	if result != nil {
		return *result, err
	}

	// == autoisitors Frontend ==========
	result, err = r.ensureDeployment(request, auto, r.frontendDeployment(auto))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, auto, r.frontendService(auto))
	if result != nil {
		return *result, err
	}

	err = r.updateFrontendStatus(auto)
	if err != nil {
		// Requeue the request
		return ctrl.Result{}, err
	}

	result, err = r.handleFrontendChanges(auto)
	if result != nil {
		return *result, err
	}

	// == Finish ==========
	// Eautoerything went fine, don't requeue
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutomationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.Automation{}).
		Complete(r)
}
