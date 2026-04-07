$ minikube start -p devx --driver=docker --nodes=3 --memory=4096m
$ minikube image load -p devx weather:latest


1. Create a kubernetes namespace for monitoring


$ kubectl create namespace monitoring

2. Add the necessary helm charts for prometheus operator

$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo update


3. Install the operator on the cluster

$ helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace --set grafana.service.port=3001 -f prometheus-values.yaml


### Headlamp install

# first add our custom repo to your local helm repositories
helm repo add headlamp https://kubernetes-sigs.github.io/headlamp/

# now you should be able to install headlamp via helm
helm install my-headlamp headlamp/headlamp --namespace kube-system

#### Port forward headlamp UI

$ kubectl port-forward -n kube-system svc/my-headlamp 8080:80

#### Create SA for headlamp

kubectl -n kube-system create serviceaccount headlamp-admin
kubectl create clusterrolebinding headlamp-admin --serviceaccount=kube-system:headlamp-admin --clusterrole=cluster-admin
kubectl create token headlamp-admin -n kube-system


4. Port forward services from the cluster

$ kubectl port-forward -n monitoring service/prometheus-operated 9090:9090
$ kubectl port-forward -n monitoring service/prometheus-operator-grafana 3001:3001


5. Install Jaeger tracing using manifests

### Install Cert Manager

$ kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.0/cert-manager.yaml

### Install Jaeger using manifests

$ kubectl create ns observability

$ wget https://github.com/jaegertracing/jaeger-operator/releases/download/v1.36.0/jaeger-operator.yaml

Edit the yaml file to change the kube-rbac-proxy image to,

# Change from:
image: gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0
# To (Example):
image: quay.io/brancz/kube-rbac-proxy:v0.15.0

$ kubectl apply -f jaeger-operator.yaml -n observability


6. Creating a Jaeger Instance

$ kubectl apply -f jaeger-allinone.yaml -n observability

$ kubectl get jaegers -n observability

$ kubectl port-forward service/my-jaeger-query 16686:16686 -n observability


### Install OpenSearch

kubectl create ns logging

helm install my-opensearch opensearch/opensearch -n logging -f opensearch-values.yaml


7. Install the weather helm chart

$ helm install weather .

$ kubectl port-forward svc/weather-weather 8081:8081


