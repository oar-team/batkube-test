#!/bin/sh

#W="../batkube/examples/workloads/KIT_10h_80.json"
#P="../batkube/examples/platforms/1node_6core.xml"
W="../batkube/examples/workloads/spaced_200_delay170.json"
P="../batkube/examples/platforms/platform_graphene_16nodes.xml"
SCHED="../../expes/kubernetes/scheduler"
KUBECONFIG="../batkube/kubeconfig.yaml"
BATKUBE="../batkube/batkube"

PASSES=20

killall batsim > /dev/null 2>&1
killall scheduler > /dev/null 2>&1
killall batkube > /dev/null 2>&1

exp_start=$(date +%s.%N)
pass=5
while [ $pass -lt $PASSES ]; do
  pass_start=$(date +%s.%N)
  echo -n "Pass $(( $pass + 1))/$PASSES..."

  echo "" > batsim.log
  echo "" > scheduler.log
  echo "" > batkube.log

  while [ "$(lsof -i -P -n | grep :27000 | wc -l)" -eq 1 ]; do
    echo e "\nWarning: port 27000 already in use. Trying to shut down the scheduler..."
    pkill scheduler
    sleep 2
  done
  $SCHED --kubeconfig="$KUBECONFIG" --kube-api-content-type=application/json --leader-elect=false --scheduler-name=default > scheduler.log 2>&1 &
  sched_pid=$!

  $BATKUBE --scheme=http --port 8001 --fast-forward-on-no-pending-jobs --detect-scheduler-deadlock --min-delay=0ms --scheduler-crash-timeout=10s --timeout-value=50ms --base-simulation-timestep=10ms --max-simulation-timestep=1000s> batkube.log 2>&1 &
  batkube_pid=$!
  sleep 5 # give time for the api to start

  out_prefix="expe-out/$(basename $W | cut -f 1 -d '.')_simu_${pass}"
  sim_start=$(date +%s.%N)
  batsim -p "$P" -w "$W" -e "$out_prefix" --enable-compute-sharing > batsim.log 2>&1 &
  batsim_pid=$!

  wait $batkube_pid
  exit_code=$?
  duration=$(echo "$(date +%s.%N) - $sim_start" | bc)

  kill $sched_pid > /dev/null 2>&1
  kill $batkube_pid > /dev/null 2>&1
  kill $batsim_pid > /dev/null 2>&1
  sleep 1 # wait for zmq to close properly

  [ $exit_code -gt 0 ] && \
    echo "Simulation failed with code $exit_code. Retrying." && \
    ((i--)) && \
    continue

  pass_duration=$(echo "$(date +%s.%N) - $pass_start" | bc)
  echo -n "Done"
  echo -n " (sim ${duration}s, total ${pass_duration}s) "
  rough_eta=$(echo "$pass_duration * ($PASSES - $pass)" | bc)
  echo "ETA: $(date -d@$rough_eta -u +%Hh%Mh%Ss) ($(date --date="$rough_eta seconds"))"

  ((pass++))
done

exp_duration=$(date -d@$(echo "$(date +%s.%N) - $exp_start" | bc) -u +%Hh%Mm%Ss)
echo "Experience lasted ${exp_duration}s"
