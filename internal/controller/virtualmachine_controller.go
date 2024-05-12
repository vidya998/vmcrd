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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	corevmoperatoriov1alpha1 "github.com/kube-works/vmoperator.git/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var i = 0

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

	log.Info("Reconciliation started for VirtualMachine", "namespace", req.Namespace, "name", req.Name)

	// Fetch the VirtualMachine instance
	vm := &corevmoperatoriov1alpha1.VirtualMachine{}
	if err := r.Get(ctx, req.NamespacedName, vm); err != nil {
		log.Error(err, "Failed to fetch VirtualMachine")
		return ctrl.Result{}, err
	}

	log.Info("Fetched VirtualMachine", "namespace", req.Namespace, "name", req.Name, "spec", vm.Spec)

	resp, err := r.GetEC2Instance(ctx, vm)
	if err != nil {
		log.Error(err, "error describing EC2 instances")
		return ctrl.Result{}, err
	}

	if vm.Status.CurrentReplicas == nil {
		vm.Status.CurrentReplicas = new(int32)
	}
	*vm.Status.CurrentReplicas = int32(len(resp.Reservations))

	if vm.Status.CurrentReplicas == nil || *vm.Status.CurrentReplicas < vm.Spec.Replicas {
		log.Info("Creating EC2 instance as current replicas are less than desired ones", "currentReplicas", vm.Status.CurrentReplicas, "desiredReplicas", vm.Spec.Replicas)
		if err := r.createEC2Instance(ctx, vm); err != nil {
			log.Error(err, "Failed to deploy EC2 instance")
			return ctrl.Result{}, err
		}
	} else if *vm.Status.CurrentReplicas > vm.Spec.Replicas {
		log.Info("Delete EC2 instances as current replicas are more than desired ones", "currentReplicas", vm.Status.CurrentReplicas, "desiredReplicas", vm.Spec.Replicas)
		// Scale down EC2 instances if there are more replicas than desired
		if err := r.deleteEC2Instances(ctx, vm); err != nil {
			log.Error(err, "Failed to delete EC2 instances")
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: 50 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corevmoperatoriov1alpha1.VirtualMachine{}).
		Complete(r)
}

func (r *VirtualMachineReconciler) createEC2Instance(ctx context.Context, vm *corevmoperatoriov1alpha1.VirtualMachine) error {

	log := log.FromContext(ctx)

	log.Info("Creating a new Ec2 instance", "instance", "ec2")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(vm.Spec.Region),
		Credentials: credentials.NewStaticCredentials("AKIA3HYEUERMNCFML6EK", "NyUziu7CQNh7jxoMq9yZKna8VoZSw/pYFs4Rr2tD", ""),
	})
	if err != nil {
		return err
	}

	svc := ec2.New(sess)

	params := &ec2.RunInstancesInput{
		ImageId:      aws.String(vm.Spec.Template),
		InstanceType: aws.String(vm.Spec.MachineType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Spec.InstanceName),
					},
				},
			},
		},
	}

	resp, err := svc.RunInstances(params)
	if err != nil {
		return err
	}

	describeResponse, err := r.GetEC2Instance(ctx, vm)
	if err != nil {
		log.Error(err, "error getting ec2")
	}

	vm.Status.InstanceID = *resp.Instances[0].InstanceId
	vm.Status.IsRunningPhase = "Running"
	vm.Status.IsPendingPhase = ""
	vm.Status.IsErrorPhase = ""
	vm.Status.DesiredReplicas = vm.Spec.Replicas
	if vm.Status.CurrentReplicas == nil {
		vm.Status.CurrentReplicas = new(int32)
	}
	*vm.Status.CurrentReplicas = int32(len(describeResponse.Reservations))
	log.Info("Number of current replicas", "current-replicas", vm.Status.CurrentReplicas)

	return r.Status().Update(ctx, vm)
}

func (r *VirtualMachineReconciler) deleteEC2Instances(ctx context.Context, vm *corevmoperatoriov1alpha1.VirtualMachine) error {
	log := log.FromContext(ctx)

	log.Info("Terminating extra EC2 replicas", "vm", vm.Name)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(vm.Spec.Region),
	})
	if err != nil {
		log.Error(err, "failed to create AWS session")
		return err
	}

	svc := ec2.New(sess)

	resp, err := r.GetEC2Instance(ctx, vm)
	if err != nil {
		log.Error(err, "error getting EC2 instances")
		return err
	}

	instancesToTerminate := len(resp.Reservations) - int(vm.Spec.Replicas)
	if instancesToTerminate <= 0 {
		log.Info("No extra replica to terminate")
		return nil
	}

	log.Info("Replicas to terminate", "count", instancesToTerminate)

	// Terminate excess instances
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			if instancesToTerminate > 0 {
				log.Info("Terminating EC2 instance", "instanceId", *instance.InstanceId)
				_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
					InstanceIds: []*string{instance.InstanceId},
				})
				if err != nil {
					log.Error(err, "failed to terminate instance", "instanceId", *instance.InstanceId)
					return err
				}
				log.Info("Instance terminated successfully", "instanceId", *instance.InstanceId)
				instancesToTerminate--
				*vm.Status.CurrentReplicas--
			} else {
				break
			}
		}
		if instancesToTerminate == 0 {
			break
		}
	}

	log.Info("Instance terminated successfully", "replicas", *vm.Status.CurrentReplicas)
	err = r.updateVirtualMachineStatus(ctx, vm)
	if err != nil {
		log.Error(err, "failed to update virtual machine status")
		return err
	}

	log.Info("Excess instances terminated successfully")
	return nil
}
func (r *VirtualMachineReconciler) updateVirtualMachineStatus(ctx context.Context, vm *corevmoperatoriov1alpha1.VirtualMachine) error {
	log := log.FromContext(ctx)

	if err := r.Status().Update(ctx, vm); err != nil {
		log.Error(err, "Failed to update VirtualMachine status")
		return err
	}
	return nil
}

func (r *VirtualMachineReconciler) GetEC2Instance(ctx context.Context, vm *corevmoperatoriov1alpha1.VirtualMachine) (*ec2.DescribeInstancesOutput, error) {

	log := log.FromContext(ctx)

	log.Info("Describing Ec2 Instance", "instance", "running")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(vm.Spec.Region),
	})
	if err != nil {
		return nil, err
	}

	svc := ec2.New(sess)

	tagFilters := []*ec2.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []*string{aws.String(vm.Spec.InstanceName)},
		},
	}
	instanceStateFilter := &ec2.Filter{
		Name:   aws.String("instance-state-name"),
		Values: []*string{aws.String("running"), aws.String("pending")},
	}

	input := &ec2.DescribeInstancesInput{
		Filters: append(tagFilters, instanceStateFilter),
	}

	// Call DescribeInstances
	response, err := svc.DescribeInstances(input)
	if err != nil {
		return nil, err
	}
	return response, nil
}
