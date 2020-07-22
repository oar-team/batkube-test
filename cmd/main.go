package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	job_id int = iota
	workload_name
	profile
	submission_time
	requested_number_of_resources
	requested_time
	success
	final_state
	starting_time
	execution_time
	finish_time
	waiting_time
	turnaround_time
	stretch
	allocated_resources
	consumed_energy
	metadata
	scheduled
	pulling
	pulled
	created
)

func main() {
	wlJson := flag.String("w", "", "File specifying a Batsim workload in json format")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig.yaml")

	flag.Parse()
	if *wlJson == "" || *kubeconfig == "" {
		fmt.Fprintf(os.Stderr, "usage:\n\tbatkube-test -w <workload.json> -config <kubeconfig.yaml>\n")
		os.Exit(1)
	}

	// Initialize kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Parse workload
	wl := parseFile(*wlJson)
	pods := translateJobsToPods(&wl)

	// Initialize channels
	noMoreJobs := make(chan bool)
	quit := make(chan struct{})
	events := make(chan *v1.Event)
	defer close(quit)

	// Initialize the experience
	initEventInformer(cs, events, quit)
	csvData := initialState(&wl)

	// Launch the experience
	wg := sync.WaitGroup{}
	wg.Add(2)
	// This will break in 2262-04-11 23:47:16.854775807 +0000 UTC :^)
	origin := time.Now()
	go func() {
		defer wg.Done()
		defer cleanupResources(cs)
		runResourceWatcher(csvData, noMoreJobs, events, origin)
	}()
	go func() {
		defer wg.Done()
		runPodSubmitter(cs, pods, noMoreJobs, origin)
	}()
	wg.Wait()
	computeRemainingData(csvData)
	spew.Dump(csvData)
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
func runPodSubmitter(cs *kubernetes.Clientset, pods []*v1.Pod, noMoreJobs chan bool, origin time.Time) {
	verifyJobSubmissionOrder(pods) // pods need to be ordered by submission time

	// TODO : find a way to submit jobs in parallel to reduce overhead when
	// multiple jobs are submitted at the same time. Unfortunately, doing
	// this naively is impossible due to the api's rate limiting policies.
	//
	// That said, maybe the reason this loop is is because of this rate
	// limiting in the first place. Then, nothing is to be done.
	one := int32(1)
	var zero int32
	for _, pod := range pods {
		offsettedSubTime := origin.Add(time.Duration(pod.CreationTimestamp.UnixNano()))
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
			log.Infof("job %s submitted", pod.Name)
		}
	}
	noMoreJobs <- true
}

func initialState(wl *translate.Workload) [][]string {
	// Do not change this order
	csvData := [][]string{{
		"job_id",
		"workload_name",
		"profile",
		"submission_time",
		"requested_number_of_resources",
		"requested_time",
		"success",
		"final_state",
		"starting_time",
		"execution_time",
		"finish_time",
		"waiting_time",
		"turnaround_time",
		"stretch",
		"allocated_resources",
		"consumed_energy",
		"metadata",
		"scheduled",
		"pulling",
		"pulled",
		"created",
	}}

	for i, job := range wl.Jobs {
		csvData = append(csvData, make([]string, len(csvData[0])))
		csvData[i+1][job_id] = job.Id
		csvData[i+1][workload_name] = "w0"
		csvData[i+1][profile] = job.Profile
		csvData[i+1][requested_number_of_resources] = "1"
		csvData[i+1][requested_time] = "0" // Time limit on pods is not implemented in batkube
		csvData[i+1][consumed_energy] = "-1"
		// TODO : check that pods completed successfully indeed
		csvData[i+1][final_state] = "COMPLETED_SUCCESSFULLY"
		csvData[i+1][success] = "1"

	}
	return csvData
}

func computeRemainingData(csvData [][]string) {

}

/*
Continuously watches the cluster state and writes the events to csvData
*/
func runResourceWatcher(csvData [][]string, noMoreJobs chan bool, events chan *v1.Event, origin time.Time) {
	var unfinishedJobs = len(csvData) - 1 // All the jobs are pending at this moment
	var noMoreJobsBool bool
	for unfinishedJobs > 0 || !noMoreJobsBool {
		select {
		case <-noMoreJobs:
			noMoreJobsBool = true
		case e := <-events:
			log.Infoln(e.Reason, e.InvolvedObject.Kind, e.InvolvedObject.Name)
			handleEvent(csvData, e, &unfinishedJobs, origin)
		}
	}
}

func handleEvent(csvData [][]string, event *v1.Event, unfinishedJobs *int, origin time.Time) {
	id := strings.Split(event.InvolvedObject.Name, "-")[1] // pods names
	var jobLine []string
	for _, line := range csvData {
		if line[0] == id {
			jobLine = line
		}
	}

	// Trying to use event.CreationTimestamp results in negative values
	// when considering "origin" as the time origin. Maybe the api server's
	// time is not entirely synchronized with this script's time.
	switch event.Reason {
	case "Completed":
		*unfinishedJobs--
		jobLine[finish_time] = timeToBatsimTime(time.Now(), origin)
	case "SuccessfulCreate":
		// Event originating from a Job
		jobLine[submission_time] = timeToBatsimTime(time.Now(), origin)
	case "Scheduled":
		// TODO get pod nodeName
		jobLine[scheduled] = timeToBatsimTime(time.Now(), origin)
	case "Pulling":
		jobLine[pulling] = timeToBatsimTime(time.Now(), origin)
	case "Pulled":
		jobLine[pulled] = timeToBatsimTime(time.Now(), origin)
	case "Started":
		jobLine[starting_time] = timeToBatsimTime(time.Now(), origin)
	case "Created":
		jobLine[created] = timeToBatsimTime(time.Now(), origin)
	default:
	}
}

func timeToBatsimTime(t time.Time, origin time.Time) string {
	return fmt.Sprintf("%f", float64(t.Sub(origin).Round(time.Millisecond))/1e9)
}

func initEventInformer(cs *kubernetes.Clientset, events chan *v1.Event, quit chan struct{}) {
	factory := informers.NewSharedInformerFactory(cs, 0)
	factory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			events <- obj.(*v1.Event)
		},
	})
	factory.Start(quit)
}

/*
Cleans up the cluster resources in preparation for the next epoch
*/
func cleanupResources(cs *kubernetes.Clientset) {
	ctx := context.Background()
	namespaces, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	var zero int64
	log.Infoln("Waiting a bit for resources to stabilize before cleaning...")
	time.Sleep(1 * time.Second)
	for _, namespace := range namespaces.Items {
		// Ignore namespaces inherent to kubernetes
		if namespace.Name == "kube-system" || namespace.Name == "kube-public" || namespace.Name == "kube-node-lease" {
			continue
		}
		if err := cs.BatchV1().Jobs(namespace.Name).DeleteCollection(ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("jobs cleaned for namespace %s", namespace.Name)
		}
		if err := cs.CoreV1().Pods(namespace.Name).DeleteCollection(ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("pods cleaned for namespace %s", namespace.Name)
		}
		if err := cs.CoreV1().Events(namespace.Name).DeleteCollection(ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("events cleaned for namespace %s", namespace.Name)
		}
	}
	log.Info("Done cleaning resources")
}
