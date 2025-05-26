# KubeDepot Helm Chart

## Basic Usage

```bash
helm repo add kubedepot https://rgeraskin.github.io/kubedepot/
helm install kubedepot kubedepot/kubedepot # This won't work without kubeconfigs
```

## Configuration

I recommend these approaches:

### 1. Use the KubeDepot Helm Chart as a Dependency

You can't work with kubeconfig files directly in the KubeDepot Helm Chart when it's used as a dependency, so you'll need to create a new root chart.

1. Create a new root chart
2. Add the KubeDepot Helm Chart as a dependency to your `Chart.yaml`:
   ```yaml
   dependencies:
     - name: kubedepot
       version: 0.1.0
       repository: https://rgeraskin.github.io/kubedepot/
   ```
3. Create your own ConfigMap template to work with kubeconfig files directly, for example:
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     labels:
       app.kubernetes.io/component: {{ .Chart.Name }}
       app.kubernetes.io/instance: {{ .Release.Name }}
       app.kubernetes.io/managed-by: Helm
       app.kubernetes.io/name: {{ .Chart.Name }}
       app.kubernetes.io/part-of: undefined
       app.kubernetes.io/version: {{ .Chart.AppVersion }}
       helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
     name: {{ .Chart.Name }}
     namespace: {{ .Release.Namespace }}
   data:
     {{- range $path, $_ := .Files.Glob "_kubeconfigs/*.yaml" }}
     {{ base $path }}: |-
     {{- $.Files.Get $path | nindent 4 }}
     {{- end }}
   ```
4. Place your kubeconfig files in the `_kubeconfigs` directory

### 2. Create Your Own KubeDepot Helm Chart

The easiest way is to use [HULL - Helm Uniform Layer Library](https://github.com/vidispine/hull):

1. Prepare your `Chart.yaml`
1. Copy and paste `hull.yaml` from the KubeDepot Helm Chart templates dir
1. Copy and paste values from the KubeDepot Helm Chart
1. Use the commented sections as a starting point

   For example, you can place kubeconfig files in the `_kubeconfigs` directory and reference them in the `values.yaml` file with `path: _kubeconfigs/clusterXXX.yaml` in the configmap section.

### 3. Use Inline Kubeconfigs in Helm Values

This is the most straightforward approach. See the commented inline configuration for the configmap section in the `values.yaml` file. The extracted values will look like this:

```yaml
hull:
  objects:
    configmap:
      kubedepot:
        data:
          cluster1.yaml:
            inline:
              apiVersion: v1
              kind: Config
              clusters:
                - name: cluster2
                  cluster:
                    server: https://api.cluster2.example.com:6443
                    certificate-authority-data: certificate-authority-data
              contexts:
                - name: cluster2-context
                  context:
                    cluster: cluster2
                    user: cluster2-admin
              users:
                - name: cluster2-admin
                  user:
                    client-certificate-data: client-certificate-data
                    client-key-data: client-key-data
              current-context: cluster2-context
```

Use this as extra values when installing the chart, or as dependency values if you're using the KubeDepot Helm Chart as a dependency.