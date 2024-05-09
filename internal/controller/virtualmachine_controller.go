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
	corevmoperatoriov1alpha1 "github.com/kube-works/vmoperator.git/api/v1alpha1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.vmoperator.io.core.vmoperator.io,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.vmoperator.io.core.vmoperator.io,resources=virtualmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.vmoperator.io.core.vmoperator.io,resources=virtualmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	// Fetch the VirtualMachine instance
	vm := &corevmoperatoriov1alpha1.VirtualMachine{}
	if err := r.Get(ctx, req.NamespacedName, vm); err != nil {
		log.Error(err, "Failed to fetch VirtualMachine")
		return ctrl.Result{}, err
	}

	// Ensure that the desired number of replicas is running
	if vm.Status.CurrentReplicas == nil || *vm.Status.CurrentReplicas < vm.Spec.Replicas {
		if err := r.deployEC2Instance(ctx, vm); err != nil {
			log.Error(err, "Failed to deploy EC2 instance")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corevmoperatoriov1alpha1.VirtualMachine{}).
		Complete(r)
}

func (r *VirtualMachineReconciler) deployEC2Instance(ctx context.Context, vm *corevmoperatoriov1alpha1.VirtualMachine) error {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(vm.Spec.Region),
	})
	if err != nil {
		return err
	}

	// Create EC2 service client
	svc := ec2.New(sess)

	// Define EC2 instance parameters based on VirtualMachine spec
	params := &ec2.RunInstancesInput{
		ImageId:      aws.String(vm.Spec.Template),
		InstanceType: aws.String(vm.Spec.MachineType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		// Add other parameters like key pair, security group, etc.
	}

	resp, err := svc.RunInstances(params)
	if err != nil {
		return err
	}

	// Update status with instance details
	vm.Status.InstanceID = *resp.Instances[0].InstanceId
	vm.Status.IsRunningPhase = "Running"
	vm.Status.IsPendingPhase = ""
	vm.Status.IsErrorPhase = ""
	vm.Status.DesiredReplicas = vm.Spec.Replicas

	return r.Status().Update(ctx, vm)
}