#!/bin/sh

# Amount of processing units and memory in the system
CPU_HOST="$(nproc)"
MEM_HOST="$(cat /proc/meminfo | head -n 1 | awk '{ print $2 }')"
K3S_TOKEN=${RANDOM}${RANDOM}${RANDOM}

help () {
  cat << EOF
  Usage: ./startup-cluster.sh -n [nodes] --cpu [cpu] --mem [mem]

  nodes: Number of desired nodes in the system
  cpu: Available compute resources for each node
  mem: Available memory for each node, in Kb
EOF
exit
}


while [ $# -gt 0 ]
do
  case $1 in
    -n)
      NODES="$2"
      shift
      shift
      ;;
    --cpu)
      CPU="$2"
      shift
      shift
      ;;
    --mem)
      MEM="$2"
      shift
      shift
      ;;
    *)
      help
      ;;
  esac
done

if [ -z $NODES ] || [ -z $CPU ] || [ -z $MEM ]
then
  echo "-n needs a plain number"
  exit
fi
if [ -z $CPU ] || [ -z $CPU ] || [ -z $MEM ]
then
  echo "--cpu needs a plain number"
  exit
fi
if [ -z $MEM ] || [ -z $CPU ] || [ -z $MEM ]
then
  echo "--mem needs a plain number. Given memory is in Kb."
  exit
fi

if [ $CPU -gt $CPU_HOST ]
then
  echo "Cannot request a number of compute units greater than host's number of cpu.
  Host cpu capacity : $CPU_HOST"
  exit
fi
if [ $MEM -gt $MEM_HOST ]
then
  echo "Cannot request a number of memory greater than host's memory capacity.
  Host memory capacity : $MEM_HOST"
  exit
fi

a=`echo $NODES | tr -d "[0-9]"`
if [ ! -z $a ]
then 
  help
fi

if [ -f "./docker-compose.yaml" ]
then
  echo "A docker-compose.yaml file already exists in this directory. Please delete it or move to another directory"
  exit
fi
touch docker-compose.yaml

echo "version: '3'
services:

  server:
    image: \"rancher/k3s:\${K3S_VERSION:-latest}\"
    command: server --disable-agent
    tmpfs:
    - /run
    - /var/run
    privileged: true
    environment:
    - K3S_TOKEN=$K3S_TOKEN
    - K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml
    - K3S_KUBECONFIG_MODE=666
    volumes:
    - k3s-server:/var/lib/rancher/k3s
    # This is just so that we get the kubeconfig file out
    - .:/output
    ports:
    - 6443:6443">> ./docker-compose.yaml

i=0
while [ $i -lt $NODES ]
do

  echo "
  agent$i:
    image: \"rancher/k3s:\${K3S_VERSION:-latest}\"
    command: agent --kubelet-arg \"system-reserved=cpu=$(($CPU_HOST - $CPU)),memory=$(($MEM_HOST - $MEM))Ki\"
    tmpfs:
    - /run
    - /var/run
    privileged: true
    environment:
    - K3S_URL=https://server:6443
    - K3S_TOKEN=$K3S_TOKEN">> ./docker-compose.yaml
    i=$(($i + 1 ))
done

echo "
volumes:
  k3s-server: {}" >> ./docker-compose.yaml

docker-compose up -d && echo "
Kubernetes cluster up and running. Try it with kubectl: \"KUBECONFIG=kubeconfig.yaml kubectl get nodes\".
Note that it may take a few minutes to initialize.

To shut down the cluster and delete the images: docker-compose down --rmi all"
