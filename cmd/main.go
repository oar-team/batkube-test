package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	wlJson := flag.String("w", "", "File specifying a Batsim workload in json format")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig.yaml")

	flag.Parse()
	if *wlJson == "" || *kubeconfig == "" {
		fmt.Fprintf(os.Stderr, "usage:\n\tbatkube-test -w <workload.json> -config <kubeconfig.yaml>\n")
		os.Exit(1)
	}

	// Initialise kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Parse workload, submit the pods
	wl := parseFile(*wlJson)
	pods := translateJobsToPods(&wl)
	submitJobs(cs, pods)
}

/*
Parses a workload file into a byte array
*/
func parseFile(file string) translate.Workload {
	wlFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(wlFile)
	// First step : decode into a map
	jsonData := make(map[string]interface{}, 0)
	err = decoder.Decode(&jsonData)
	if err != nil {
		log.Fatal(err)
	}

	// Second step : decode again into a workload and decode profiles.
	wl := translate.Workload{}
	if err := mapstructure.Decode(jsonData, &wl); err != nil {
		log.Fatal(err)
	}
	for profileName, profileData := range jsonData["profiles"].(map[string]interface{}) {
		profile := wl.Profiles[profileName]
		profileDataMap := profileData.(map[string]interface{})

		// Profile specs
		if err := mapstructure.Decode(profileDataMap, &profile.Specs); err != nil {
			log.Fatal(err)
		}
		delete(profile.Specs, "type")
		delete(profile.Specs, "ret")
		wl.Profiles[profileName] = profile
	}
	return wl
}

func translateJobsToPods(wl *translate.Workload) []*v1.Pod {
	simData := translate.SimulationBeginsData{
		Profiles: make(map[string]map[string]translate.Profile, 0),
	}
	// Only one workload for now
	simData.Profiles["w0"] = make(map[string]translate.Profile, 0)
	for profileName, profile := range wl.Profiles {
		simData.Profiles["w0"][profileName] = profile
	}

	pods := make([]*v1.Pod, 0)
	for _, job := range wl.Jobs {
		// JobToPod takes as input a JOB_SUBMITTED event, where the id is workload!jobId.
		job.Id = "w0!" + job.Id
		err, pod := translate.JobToPod(job, simData)
		if err != nil {
			log.Fatal(err)
		}

		// models.IoK8sAPICoreV1Pod -> v1.Pod
		// Translation is done thanks to the json tags that remain the same.
		corev1Pod := v1.Pod{}
		podBytes, err := json.Marshal(pod)
		if err != nil {
			log.Fatal(err)
		}
		if err = json.Unmarshal(podBytes, &corev1Pod); err != nil {
			log.Fatal(err)
		}
		pods = append(pods, &corev1Pod)
	}
	return pods
}

func verifyJobSubmissionOrder(pods []*v1.Pod) {
	lastCreationTimeStamp := time.Unix(0, 0)
	for _, pod := range pods {
		if pod.CreationTimestamp.Time.Before(lastCreationTimeStamp) {
			log.Fatal("pods are not ordered by submission timestamps")
		}
		lastCreationTimeStamp = pod.CreationTimestamp.Time
	}
}

/*
Submits the given jobs at the correct timestamps.
*/
func submitJobs(cs *kubernetes.Clientset, pods []*v1.Pod) {
	verifyJobSubmissionOrder(pods)

	origin := time.Now()
	log.Infof("[%s] Experience starts")
	offset := origin.Sub(time.Unix(0, 0))
	// TODO : do that with parallel tasks to minimize overhead when
	// multiple jobs are submitted at the same time.
	for _, pod := range pods {
		offsettedSubTime := pod.CreationTimestamp.Add(offset)
		if time.Now().Before(offsettedSubTime) {
			time.Sleep(offsettedSubTime.Sub(time.Now()))
		}
		cs.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
		log.Infof("[%s] job %s submitted", time.Now(), pod.Name, pod.CreationTimestamp.Time)
	}
}
