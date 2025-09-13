$ minikube start --driver=docker --nodes=3
$ minikube image load weather:latest


1. Create a kubernetes namespace for monitoring


$ kubectl create namespace monitoring

2. Add the necessary helm charts for prometheus operator

$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo update


3. Install the operator on the cluster

$ helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace --set grafana.service.port=3001

4. Port forward required services from the cluster

$ kubectl port-forward svc/weather 8080:8080
$ kubectl port-forward -n monitoring service/prometheus-operator-kube-p-prometheus 9090:9090
$ kubectl port-forward -n monitoring service/prometheus-operator-grafana -n monitoring 3001:3001
