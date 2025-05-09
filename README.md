# warrior-operator

Kubernetes operator for the [ArchiveTeam Warrior](https://wiki.archiveteam.org/index.php/ArchiveTeam_Warrior) `-grab`
container images.

## Description

Allows easy deployment of ArchiveTeam Warriors to your cluster, to help the efforts of archiving the web. Define a 
warrior and get going!

```yaml
apiVersion: "warrior.k8s.gmem.ca/v1alpha1"
kind: Warrior
metadata:
  name: twitch
spec:
  project: twitch # Project to work on
  downloader: arch-dog # Your own downloader name for credit!
  scaling:
    minimum: 0 # Minimum number of pods to run
    maximum: 5 # Maximum number of pods to run
    concurrency: 5 # Concurrent jobs per pod.
  resources: # Resource limits and requests for the pods
    cacheSize: 500Mi # Size of the memory cache volume
    limits:
      cpu: "1"
      memory: "1Gi"
    requests:
      cpu: "1m"
      memory: "256Mi"
```

## Installing

```sh
# Forgejo
kubectl apply -f https://git.gmem.ca/arch/warrior-operator/raw/branch/main/dist/install.yaml
# GitHub
kubectl apply -f https://raw.githubusercontent.com/gmemstr/warrior-operator/refs/heads/main/dist/install.yaml
```

Images are mirrored to ghcr.io.

* `git.gmem.ca/arch/warrior-operator:latest`
* `ghcr.io/gmemstr/warrior-operator:latest`

## Contributing

Pull requests and issues are accepted [on GitHub](https://github.com/gmemstr/warrior-operator) or via email.

### Getting Started

#### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

#### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/warrior-operator:tag
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
make deploy IMG=<some-registry>/warrior-operator:tag
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
## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

