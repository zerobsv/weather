1. Create a kubernetes namespace for monitoring

$ kubectl create namespace monitoring

2. Add the necessary helm charts for prometheus operator

$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo update


3. Install the operator on the cluster

$ helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack -n monitoring
