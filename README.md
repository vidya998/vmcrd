# vmoperator
 Kubernetes controller based on a simple CRD which will deploy an ec2 instance in AWS cloud and manage its lifecycle.

## Description
 This project presents a simple CRD (Custom Resource Definition) for creating EC2 instances in the AWS cloud, accompanied by a controller with reconciliation logic. The controller ensures that the desired number of instances is maintained, spinning up new instances when the current replicas fall below the desired count and deleting extra instances when there are more replicas than desire 

## Specifications
- CPU: Specifies the CPU configuration.
- Memory: Defines memory requirements.
- Template: Describes the instance template.
- Replicas: Sets the number of replicas.
- Host OS: Specifies the operating system.
- Machine Type: Defines the machine type.
- Region: Specifies the AWS region.
- Instance Name: Provides a unique name for the instance.
## Statuses
- InstanceID: Unique identifier for each instance.
- IsRunningPhase: Indicates if the instance is running.
- IsPendingPhase: Indicates if the instance creation is pending.
- IsErrorPhase: Indicates if there's an error with the instance.
- CurrentReplicas: Number of currently running instances.
- DesiredReplicas: Number of instances desired.
## Deployment
The controller is deployed in a local-cluster using Minikube. A Docker image is created and pushed to Docker Hub for accessibility.
[Docker Hub Link](https://hub.docker.com/layers/ividyaverma998/vmoperator/v2/images/sha256:ef0404c5caf4aa4a59c00108a734182d0a3e848518d299abc0719bab82511676?uuid=eb63b97a-bb21-4204-8562-242beb0d33f8%0A)

## Helmfile
A Helmfile is included with all specifications required for creating the resource. The controller listens for any resource creation events of kind VirtualMachine.

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=docker.io/ividyaverma998/vmoperator:v2
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=docker.io/ividyaverma998/vmoperator:v2 
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/vmoperator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/vmoperator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

