/*
Copyright 2024.

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
	ociv1beta1 "github.com/btwseeu78/mirror-image/api/v1beta1"
	"github.com/btwseeu78/mirror-image/utility"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

// ReplicationReconciler reconciles a Replication object
type ReplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=oci.platform.io,resources=replications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.platform.io,resources=replications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.platform.io,resources=replications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Replication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("oci-controller", req.NamespacedName)

	// TODO(user): your logic here
	oci := &ociv1beta1.Replication{}
	err := r.Get(ctx, req.NamespacedName, oci)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Replication Not found its might be getting deleted")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to fetch oci")
		return ctrl.Result{}, err
	}
	//
	//ociSyncList := &ociv1beta1.ReplicationList{}
	//err = r.List(ctx, ociSyncList)
	//if err != nil {
	//	log.Error(err, "unable to list oci")
	//	return ctrl.Result{}, err
	//}

	// get sync duration
	duration, err := time.ParseDuration(oci.Spec.PollingInterval)
	if err != nil {
		log.Error(err, "unable to parse duration")
		return ctrl.Result{}, err
	}
	// Get the current time
	currentTime := metav1.Now()
	if !oci.Status.LastSyncTime.IsZero() {
		// Calculate Next Sync Time
		nextSyncTime := oci.Status.LastSyncTime.Add(duration)
		if currentTime.After(nextSyncTime) {
			log.Info("Syncing Image")
			// Get the required details
			sRepo, err := utility.NewRepository(oci.Spec.SourceImage.RepoUrl)
			if err != nil {
				log.Error(err, "unable to create source repository")
				return ctrl.Result{}, err
			}

			//dRepo, err := utility.NewRepository(oci.Spec.DestinationImage.RepoUrl)
			//if err != nil {
			//	log.Error(err, "unable to create destination repository")
			//	return ctrl.Result{}, err
			//}

			// Get The Keychain Information
			sSecret := oci.Spec.SourceImage.SecretName
			dSecret := oci.Spec.DestinationImage.SecretName

			// get the actual secret
			kc, err := r.GenerateK8sChain(ctx, sSecret, dSecret, oci.Namespace)
			if err != nil {
				log.Error(err, "unable to create keychain")
				return ctrl.Result{}, err
			}

			ListRepositoryTags, err := utility.ListRepositoryTags(sRepo, kc, oci.Spec.FilterCriteria)
			if err != nil {
				log.Error(err, "unable to list source repository tags")
				return ctrl.Result{}, err
			}
			if len(ListRepositoryTags) == 0 {
				log.Info("No tags found in the source repository")
			}
			log.Info("Source Repository Tags", "Tags", ListRepositoryTags)

			oci.Status.LastSyncTime = currentTime
			err = r.Status().Update(ctx, oci)
			if err != nil {
				log.Error(err, "unable to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: duration}, nil
		} else {
			log.Info("Not Syncing Image the time is not up")
			return ctrl.Result{RequeueAfter: nextSyncTime.Sub(currentTime.Time)}, nil
		}
	}
	log.Info("Sync in progress curr time")
	oci.Status.LastSyncTime = currentTime
	err = r.Status().Update(ctx, oci)

	if err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.Replication{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *ReplicationReconciler) GenerateK8sChain(ctx context.Context, sSecret, dSecret string, secretNamespace string) (authn.Keychain, error) {
	// get the actual secret
	sPullSecret := &v1.Secret{}
	dPullSecret := &v1.Secret{}
	pullSecretCombined := make([]v1.Secret, 0)
	if sSecret == "" && dSecret == "" {
		return nil, nil
	}
	if sSecret != "" {
		err := r.Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: sSecret}, sPullSecret)
		if err != nil {
			return nil, err
		}
	}
	if dSecret != "" {
		err := r.Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: dSecret}, dPullSecret)
		if err != nil {
			return nil, err
		}
	}
	if dSecret == "" {
		pullSecretCombined = append(pullSecretCombined, *sPullSecret)
	} else if sSecret == "" {
		pullSecretCombined = append(pullSecretCombined, *dPullSecret)
	} else {

		pullSecretCombined = []v1.Secret{*sPullSecret, *dPullSecret}
	}
	// create keychain out of it
	sKeyChain, err := k8schain.NewFromPullSecrets(ctx, pullSecretCombined)
	if err != nil {
		return nil, err
	}
	return sKeyChain, nil
}
