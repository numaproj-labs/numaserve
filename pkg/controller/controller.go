/*
Copyright 2023 sarabala1979.

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

	"github.com/numaproj-labs/numaserve/api/v1alpha1"
	"github.com/numaproj/numaflow/pkg/shared/logging"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	numaflowV1alpha1 "github.com/numaproj/numaflow/pkg/apis/numaflow/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logging.NewLogger().Named("MLInferenceReconciler")

// MLInferenceReconciler reconciles a MLInference object
type MLInferenceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Cache      cache.Cache
	RESTMapper meta.RESTMapper
}

//+kubebuilder:rbac:groups=mlserve.numaproj.io,resources=mlinferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mlserve.numaproj.io,resources=mlinferences/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mlserve.numaproj.io,resources=mlinferences/finalizers,verbs=update
//+kubebuilder:rbac:groups=numaflow.numaproj.io,resources=pipelines;interstepbufferservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MLInference object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile

func (r *MLInferenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log.Log.Info(req.String())
	var obj v1alpha1.MLInference
	err := r.Get(ctx, req.NamespacedName, &obj)
	//log.Log.Info("%v", err)
	if err != nil {
		log.Log.Error(err, "Error occurred in processing object ", req.Namespace)
	}
	mliCtx := NewMLInferenceOperateContext(ctx, log.Log, r.Client, &obj)
	// TODO needs to refactor after POC
	//go func() {
	err = mliCtx.Process()
	if err != nil {
		log.Log.Error(err, "error reconciling", "namespace", obj.Namespace, "name", obj.Name)
	}
	//}()

	// TODO(user): your logic here
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MLInferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//Todo currently reconciler will handle only create/update, We need to find how to handle the delete

	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				switch e.Object.(type) {
				case *v1alpha1.MLInference:
					// The reconciler adds a finalizer so we perform clean-up
					// when the delete timestamp is added
					// Suppress Delete events to avoid filtering them out in the Reconcile function
					return false
				}
				return true
			},
		}).For(&v1alpha1.MLInference{}).
		Owns(&appv1.StatefulSet{}).
		Owns(&numaflowV1alpha1.Pipeline{}).
		Owns(&numaflowV1alpha1.InterStepBufferService{}).
		/*Watches( // should probably remove but this also seemed to work so leaving it in for reference
			//source.Kind(cache.Cache{}, &appv1.StatefulSet{}),
			&appv1.StatefulSet{},
			handler.EnqueueRequestForOwner(r.Scheme, r.RESTMapper, &v1alpha1.MLInference{}, handler.OnlyControllerOwner()),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(ue event.UpdateEvent) bool { return true },
				DeleteFunc: func(de event.DeleteEvent) bool { return true },
			}),
		).*/
		Complete(r)
}
