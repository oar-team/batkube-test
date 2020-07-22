# batkube test

This is a repository to conduct tests on a real cluster, in order to validate
Batkube's results.

## Dependencies

* docker-compose

## Usage

To launch a new cluster : `./scripts/startup-cluster.sh -n <number of desired
nodes> --cpu <cpu per node> --mem <memory per node (in kilobytes)>`

To run an experiment : `go run cmd/main.go -w workload.json -kubeconfig
kubeconfig.yaml -out output/dir/prefix -epochs n` with n being the number of
iterations of the same experiment you desire to make
