#!/bin/sh

#W="../batkube/examples/workloads/KIT_10h_80.json"
#P="../batkube/examples/platforms/1node_6core.xml"
W="../batkube/examples/workloads/spaced_200_delay170.json"
P="../batkube/examples/platforms/platform_graphene_16nodes.xml"
SCHED="../../expes/kubernetes/scheduler"
KUBECONFIG="../batkube/kubeconfig.yaml"
BATKUBE="../batkube/batkube"

RESUME=false

END_EXPONENT=5 # 100s (1e5ms)
# logarithmic scale

out="expe-out/max-timestep_$(basename $W | cut -f 1 -d '.').csv"

if [ $RESUME = true ]; then
  echo "Resuming experiment on $out"
else
  if [ -f "$out" ]; then
    echo "$out already exists."
    read -p "Overwrite? [Y/n] " input
    if ! [ \( -z "$input" \) -o \( "$input" = "Y" -o "$input" = "y" \) ]
    then
      echo "exiting"
      exit
    fi
  fi
  echo "max_timestep,duration,makespan,mean_waiting_time" > $out
fi

killall batsim > /dev/null 2>&1
killall scheduler > /dev/null 2>&1
killall batkube > /dev/null 2>&1

e=0
exp_start=$(date +%s.%N)
while [ $e -lt $END_EXPONENT ]; do
  i=0
  echo "Current exponent: $e/$END_EXPONENT"
  while [ $i -lt 10 ]; do
    ((i++))
    if [ -z $zeros ]; then
      continue
    fi
    max_timestep="${i}${zeros}"

    step_start=$(date +%s.%N)
    echo -n "max timestep=${max_timestep}ms..."

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

    $BATKUBE --scheme=http --port 8001 --fast-forward-on-no-pending-jobs --detect-scheduler-deadlock --min-delay=0ms --scheduler-crash-timeout=30s --timeout-value=50ms --max-simulation-timestep=${max_timestep}ms> batkube.log 2>&1 &
    batkube_pid=$!
    sleep 5 # give time for the api to start

    sim_start=$(date +%s.%N)
    batsim -p "$P" -w "$W" -e "expe-out/max-timestep" --enable-compute-sharing > batsim.log 2>&1 &
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
      continue

    res=$(tail -n 2 batsim.log | head -n 1)
    makespan=$(echo $res | awk '{ print $3 }' | grep -Eo '[0-9]+([.][0-9]+)?')
    mean_waiting_time=$(echo $res | awk '{ print $5 }' | grep -Eo '[0-9]+([.][0-9]+)?')

    echo -n "Done"

    echo "${max_timestep},${duration},${makespan},${mean_waiting_time}" >> $out

    step_duration=$(echo "$(date +%s.%N) - $step_start" | bc)
    echo " (sim ${duration}s, total ${step_duration}s)"
  done
  zeros="${zeros}0"
  ((e++))
done

exp_duration=$(date -d@$(echo "$(date +%s.%N) - $exp_start" | bc) -u +%Hh%Mm%Ss)
echo "Experience lasted ${exp_duration}s"
echo "Writing output to $out"
