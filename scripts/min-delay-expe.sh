#!/bin/sh

W=../batkube/examples/workloads/spaced_200_delay170.json
P=../batkube/examples/platforms/platform_graphene_16nodes.xml
SCHED=../../expes/kubernetes/scheduler
KUBECONFIG=../batkube/kubeconfig.yaml
BATKUBE=../batkube/batkube

# min delay starting and ending values in ms
START=0
END=100
N=50 # number of points to compute
PASSES=10 # number of trials per point

OUT="expe-out/min-delay-$(basename $W | cut -f 1 -d '.').csv"

if [ -f "$OUT" ]; then
  echo "$OUT already exists."
  read -p "Overwrite? [Y/n] " input
  if ! [ \( -z "$input" \) -o \( "$input" = "Y" -o "$input" = "y" \) ]
  then
    echo "exiting"
    exit
  fi
  echo
fi


touch "$OUT"
echo "delay success_rate mean_failure_sim_time mean_success_sim_time" > $OUT

killall batsim > /dev/null 2>&1
killall scheduler > /dev/null 2>&1
killall batkube > /dev/null 2>&1

step=$(( ($END-$START)/$N ))

delay=$START
j=0
exp_start=$(date +%s.%N)
while [ $delay -lt $END ]; do

  echo "=======min-delay = $delay (from $START to $END, step $j out of $N)======="
  echo

  successes=0
  total_success_sim_time=0
  total_failure_sim_time=0

  i=0
  step_start=$(date +%s.%N)
  while [ $i -lt $PASSES ]; do
    echo "" > batsim.log
    echo "" > scheduler.log
    echo "" > batkube.log

    echo "Pass $(( $i + 1 )) out of $PASSES"

    echo "Launching Batkube and the scheduler"
    $SCHED --kubeconfig="$KUBECONFIG" --kube-api-content-type=application/json --leader-elect=false --scheduler-name=default > scheduler.log 2>&1 &
    sched_pid=$!

    $BATKUBE --scheme=http --port 8001 --fast-forward-on-no-pending-jobs --detect-scheduler-deadlock --min-delay "$delay"ms > batkube.log 2>&1 &
    batkube_pid=$!
    sleep 4 # give time for the api to start

    echo "Simulation starts"
    start=$(date +%s.%N)
    batsim -p "$P" -w "$W" -e "expe-out/min-delay" --enable-compute-sharing > batsim.log 2>&1 &
    batsim_pid=$!

    wait $batkube_pid
    exit_code=$?
    duration=$(echo "$(date +%s.%N) - $start" | bc)

    echo "Result: exit code $exit_code; simulation time $duration"

    kill $sched_pid > /dev/null 2>&1
    kill $batkube_pid > /dev/null 2>&1
    kill $batsim_pid > /dev/null 2>&1

    if [ $exit_code -gt 1 ]; then
      echo "Unexpected exit code from Batkube: $exit_code. Retrying."
      echo
      continue
    fi
    ((successes+= 1 - exit_code))
    if [ $exit_code -eq 1 ]; then
      total_failure_sim_time=$(echo "scale=3; $total_failure_sim_time + $duration" | bc)
    else
      total_success_sim_time=$(echo "scale=3; $total_success_sim_time + $duration" | bc)
    fi

    ((i++))
    echo
  done

  step_duration=$(echo "$(date +%s.%N) - $step_start" | bc)
  success_rate=$(echo "scale=2; $successes / $PASSES" | bc)
  if [ $successes -gt 0 ]; then
    mean_success_sim_time=$(echo "scale=3; $total_success_sim_time / $successes" | bc)
  else
    mean_success_sim_time=0
  fi
  mean_failure_sim_time=$(echo "scale=3; $total_failure_sim_time / ($PASSES-$successes)" | bc)
  ((j++))
  echo "$delay $success_rate $mean_failure_sim_time $mean_success_sim_time" >> $OUT

  echo "Step done in $step_duration s"
  echo "Success rate $success_rate"
  echo "Avg success sim time $mean_success_sim_time"
  echo "Avg failure sim time $mean_failure_sim_time"

  ((delay+=$step))
  echo
done

exp_duration=$(echo "$(date +%s.%N) - $exp_start" | bc)
echo "Experience lasted $exp_duration s"
