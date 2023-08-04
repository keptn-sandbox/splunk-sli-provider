# splunk-sli-provider

Helm Chart for the keptn splunk-sli-provider

## Configuration

The following table lists the configurable parameters of the splunk-sli-provider chart and their default values.

| Parameter                               | Description                                                  | Default                                       |
| --------------------------------------- | ------------------------------------------------------------ | --------------------------------------------- |
| `splunkservice.image.repository`        | Container image name                                         | `"ghcr.io/keptn-sandbox/splunk-sli-provider"` |
| `splunkservice.image.pullPolicy`        | Kubernetes image pull policy                                 | `"IfNotPresent"`                              |
| `splunkservice.image.tag`               | Container tag                                                | `""`                                          |
| `splunkservice.existingSecret`          | Use an existing secret in k8s                                | `""`                                          |
| `spHost`                                | Define the host of the splunk instance                       | `""`                                          |
| `spPort `                               | Define the port of the splunk instance                       | `""`                                          |
| `spUsername `                           | Define the username of the splunk instance                   | `""`                                          |
| `spPassword `                           | Define the password of the splunk instance                   | `""`                                          |
| `spApitoken `                           | Define the token of the splunk instance                      | `""`                                          |
| `spSessionKey`                          | Define the session key of the splunk instance                | `""`                                          |
| `splunkservice.service.enabled`         | Creates a kubernetes service for the splunk-sli-provider     | `true`                                        |
| `distributor.stageFilter`               | Sets the stage this helm service belongs to                  | `""`                                          |
| `distributor.serviceFilter`             | Sets the service this helm service belongs to                | `""`                                          |
| `distributor.projectFilter`             | Sets the project this helm service belongs to                | `""`                                          |
| `distributor.image.repository`          | Container image name                                         | `"ghcr.io/keptn/distributor"`                 |
| `distributor.image.pullPolicy`          | Kubernetes image pull policy                                 | `"IfNotPresent"`                              |
| `distributor.image.tag`                 | Container tag                                                | `""`                                          |
| `remoteControlPlane.enabled`            | Enables remote execution plane mode                          | `false`                                       |
| `remoteControlPlane.api.protocol`       | Used protocol (http, https)                                  | `"https"`                                     |
| `remoteControlPlane.api.hostname`       | Hostname of the control plane cluster (and port)             | `""`                                          |
| `remoteControlPlane.api.apiValidateTls` | Defines if the control plane certificate should be validated | `true`                                        |
| `remoteControlPlane.api.token`          | Keptn api token                                              | `""`                                          |
| `imagePullSecrets`                      | Secrets to use for container registry credentials            | `[]`                                          |
| `serviceAccount.create`                 | Enables the service account creation                         | `true`                                        |
| `serviceAccount.annotations`            | Annotations to add to the service account                    | `{}`                                          |
| `serviceAccount.name`                   | The name of the service account to use.                      | `""`                                          |
| `podAnnotations`                        | Annotations to add to the created pods                       | `{}`                                          |
| `podSecurityContext`                    | Set the pod security context (e.g. fsgroups)                 | `{}`                                          |
| `securityContext`                       | Set the security context (e.g. runasuser)                    | `{}`                                          |
| `resources`                             | Resource limits and requests                                 | `{}`                                          |
| `nodeSelector`                          | Node selector configuration                                  | `{}`                                          |
| `tolerations`                           | Tolerations for the pods                                     | `[]`                                          |
| `affinity`                              | Affinity rules                                               | `{}`                                          |
