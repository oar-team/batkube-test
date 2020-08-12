#!/bin/sh

W=../batkube/examples/workloads/KIT_10h_80.json
P=../batkube/examples/platforms/1node_6core.xml
SCHED=../../expes/kubernetes/scheduler
KUBECONFIG=../batkube/kubeconfig.yaml
BATKUBE=../batkube/batkube

# min delay starting and ending values in ms
START=0
END=50
STEP=5
PASSES=15 # number of trials per point

out="expe-out/min-delay-$(basename $W | cut -f 1 -d '.').csv"

if [ -f "$out" ]; then
  echo "$out already exists."
  read -p "Overwrite? [Y/n] " input
  if ! [ \( -z "$input" \) -o \( "$input" = "Y" -o "$input" = "y" \) ]
  then
    echo "exiting"
    exit
  fi
fi


touch "$out"
echo "delay,exit_code,duration" > $out

killall batsim > /dev/null 2>&1
killall scheduler > /dev/null 2>&1
killall batkube > /dev/null 2>&1

n=$(echo "scale=0; ($END - $START) / $STEP + 1" | bc)

delay=$START
j=0
exp_start=$(date +%s.%N)
while [ $delay -le $END ]; do

  echo -e "\n=======min-delay=${delay}ms (step $(( $j+1 ))/$n)======="

  successes=0
  total_success_sim_time=0

  echo "" > batsim.log
  echo "" > scheduler.log
  echo "" > batkube.log

  i=0
  step_start=$(date +%s.%N)
  while [ $i -lt $PASSES ]; do
    pass_start=$(date +%s.%N)
    echo -n "Pass $(( $i + 1 )) out of $PASSES..."

    while [ "$(lsof -i -P -n | grep :27000 | wc -l)" -eq 1 ]; do
      echo e "\nWarning: port 27000 already in use. Trying to shut down the scheduler..."
      pkill scheduler
      sleep 2
    done
    $SCHED --kubeconfig="$KUBECONFIG" --kube-api-content-type=application/json --leader-elect=false --scheduler-name=default > scheduler.log 2>&1 &
    sched_pid=$!

    $BATKUBE --scheme=http --port 8001 --fast-forward-on-no-pending-jobs --detect-scheduler-deadlock --scheduler-crash-timeout=30s --min-delay "$delay"ms > batkube.log 2>&1 &
    batkube_pid=$!
    sleep 5 # give time for the api to start

    start=$(date +%s.%N)
    batsim -p "$P" -w "$W" -e "expe-out/min-delay" --enable-compute-sharing > batsim.log 2>&1 &
    batsim_pid=$!

    wait $batkube_pid
    exit_code=$?
    duration=$(echo "$(date +%s.%N) - $start" | bc)

    kill $sched_pid > /dev/null 2>&1
    kill $batkube_pid > /dev/null 2>&1
    kill $batsim_pid > /dev/null 2>&1
    sleep 1 # wait for zmq to close properly

    [ $exit_code -gt 1 ] && \
      echo "Unexpected exit code from Batkube: $exit_code. Retrying." && \
      continue

    ([ $exit_code -eq 0 ] && echo -n "passed") || echo -n "failed"
    # successes=$(( $successes + 1 - $exit_code ))
    ((successes+= 1 - $exit_code))
    total_success_sim_time=$(echo "$total_success_sim_time + $duration" | bc)

    echo "${delay},${exit_code},${duration}" >> $out

    echo " (sim ${duration}s, total $(echo "$(date +%s.%N) - $pass_start" | bc)s)"
    ((i++))
  done

  step_duration=$(echo "$(date +%s.%N) - $step_start" | bc)

  echo -e "\nStep done in ${step_duration}s"
  echo "Success rate $(echo "scale=2; $successes / $PASSES" | bc)"
  [ $successes -gt 0 ] && \
    echo "Avg success simulation time $(echo "scale=3; $total_success_sim_time / $successes" | bc)"

  rough_eta=$(echo "$step_duration * ($n - $j)" | bc)
  echo -e "\nETA: $(date -d@$rough_eta -u +%Hh%Mh%Ss) ($(date --date="$rough_eta seconds"))"

  ((delay+=$STEP))
  ((j++))
done

exp_duration=$(date -d@$(echo "$(date +%s.%N) - $exp_start" | bc) -u +%Hh%Mm%Ss)
echo "Experience lasted ${exp_duration}s"
