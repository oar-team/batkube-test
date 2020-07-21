package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
	batchv1 "k8s.io/api/batch/v1"
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
	noMoreJobs := make(chan bool)
	wg := sync.WaitGroup{}
	go func() {
		defer wg.Done()
		defer cleanupResources(cs)
		wg.Add(1)
		getJobExecutionData(noMoreJobs)
	}()
	submitJobs(cs, pods, noMoreJobs)
	wg.Wait()
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
func submitJobs(cs *kubernetes.Clientset, pods []*v1.Pod, noMoreJobs chan bool) {
	verifyJobSubmissionOrder(pods)

	origin := time.Now()
	log.Infof("[%s] Experience starts", origin)
	offset := origin.Sub(time.Unix(0, 0))
	// TODO : find a way to submit jobs in parallel to reduce overhead when
	// multiple jobs are submitted at the same time. Unfortunately, doing
	// this naively is impossible due to the api's rate limiting policies.
	// Maybe even the reason this loop is so slow is because of those
	// policies, and the client waiting for the api to be ready again.
	one := int32(1)
	var zero int32
	for _, pod := range pods {
		offsettedSubTime := pod.CreationTimestamp.Add(offset)
		if time.Now().Before(offsettedSubTime) {
			time.Sleep(offsettedSubTime.Sub(time.Now()))
		}
		pod.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		if _, err := cs.BatchV1().Jobs(pod.Namespace).Create(context.Background(),
			&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name: pod.Name,
				},
				Spec: batchv1.JobSpec{
					Completions:             &one,
					TTLSecondsAfterFinished: &zero,
					Template: v1.PodTemplateSpec{
						Spec: pod.Spec,
					},
				},
			},
			metav1.CreateOptions{},
		); err != nil {
			log.Warnln(err)
		} else {
			log.Infof("[%s] job %s submitted", time.Now(), pod.Name)
		}
	}
	noMoreJobs <- true
}

/*
Continuously watches the cluster state and gets the necessary information to
generate a csv file, with the same information as Batsim csv output.

Stops watching the cluster when all jobs are terminated and upon reception of a
value on noMoreJobs.
*/
func getJobExecutionData(noMoreJobs chan bool) {
	// TODO
	time.Sleep(1 * time.Second)
	<-noMoreJobs
}

/*
Cleans up the cluster resources in preparatoin for the next epoch
*/
func cleanupResources(cs *kubernetes.Clientset) {
	ctx := context.Background()
	namespaces, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	//var zero int64
	for _, namespace := range namespaces.Items {
		if err := cs.BatchV1().Jobs(namespace.Name).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("[%s] jobs cleaned for namespace %s", time.Now(), namespace.Name)
		}
		if err := cs.CoreV1().Pods(namespace.Name).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("[%s] pods cleaned for namespace %s", time.Now(), namespace.Name)
		}
	}
	log.Info("Done cleaning resources")
}
