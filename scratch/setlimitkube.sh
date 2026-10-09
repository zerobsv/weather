#!/bin/bash

minikube ssh -p devx -n devx "echo 'fs.inotify.max_user_watches=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx "echo 'fs.inotify.max_user_instances=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx "echo 'fs.files_max=10000000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"

# Node 02
minikube ssh -p devx -n devx-m02 "echo 'fs.inotify.max_user_watches=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx-m02 "echo 'fs.inotify.max_user_instances=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx-m02 "echo 'fs.files_max=10000000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"

# Node 03
minikube ssh -p devx -n devx-m03 "echo 'fs.inotify.max_user_watches=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx-m03 "echo 'fs.inotify.max_user_instances=100000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
minikube ssh -p devx -n devx-m03 "echo 'fs.files_max=10000000' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
