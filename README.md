# WizCloudConfigurationRule

A Kubernetes operator that introduces a `WizCloudConfigurationRule` Custom Resource Definition (CRD), enabling GitOps-style management of [Wiz](https://www.wiz.io/) Cloud Configuration Rules.

## Description

`WizCloudConfigurationRule` is a Kubernetes CRD that lets platform and security teams define and manage Wiz Cloud Configuration Rules declaratively — as Kubernetes resources stored in Git. This enables standard GitOps workflows (pull requests, review, audit history) for Wiz policy management.

**Current scope:** Admission Controller rules — rules that evaluate resources at admission time against your defined security policies.

**Planned expansion:** Additional Wiz Cloud Configuration Rule types beyond Admission Controller (e.g. scheduled, event-driven).

### Example Resource

```yaml
apiVersion: security.joseberr.io/v1
kind: WizCloudConfigurationRule
metadata:
  name: my-admission-rule
spec:
  ruleName: "Require resource limits"
  description: "Ensures all pods define CPU and memory limits"
  findingSeverity: High
  projectScope: "my-wiz-project"
```

## Getting Started

### Prerequisites

- Go v1.24.6+
- Docker 17.03+
- kubectl v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Deploy to the cluster

**Build and push the controller image:**

```sh
make docker-build docker-push IMG=<some-registry>/wizcloudconfigurationrule:tag
```

**Install the CRDs:**

```sh
make install
```

**Deploy the controller:**

```sh
make deploy IMG=<some-registry>/wizcloudconfigurationrule:tag
```

> **Note:** If you encounter RBAC errors, you may need cluster-admin privileges.

**Apply sample resources:**

```sh
kubectl apply -k config/samples/
```

### Uninstall

```sh
kubectl delete -k config/samples/   # delete CRs
make uninstall                       # delete CRDs
make undeploy                        # remove the controller
```

## Distribution

### YAML bundle

```sh
make build-installer IMG=<some-registry>/wizcloudconfigurationrule:tag
kubectl apply -f dist/install.yaml
```

### Helm Chart

```sh
kubebuilder edit --plugins=helm/v2-alpha
```

The chart is generated under `dist/chart`. Re-run this command after project changes (use `--force` if you have webhooks, and manually re-apply any custom values afterwards).

## Contributing

Run `make help` for available make targets.

More information: [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.