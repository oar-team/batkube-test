package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

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
	jobIdIndex int = iota
	workloadNameIndex
	profileIndex
	submissionTimeIndex
	requestedNumberOfResourcesIndex
	requestedTimeIndex
	successIndex
	finalStateIndex
	startingTimeIndex
	executionTimeIndex
	finishTimeIndex
	waitingTimeIndex
	turnaroundTimeIndex
	stretchIndex
	allocatedResourcesIndex
	consumedEnergyIndex
	metadataIndex
	scheduledIndex
	pullingIndex
	pulledIndex
	createdIndex
)

type submitter struct {
	ctx            context.Context
	cs             *kubernetes.Clientset
	noMoreJobs     chan bool
	events         chan *v1.Event
	origin         time.Time
	unfinishedJobs int
}

func main() {
	wlJson := flag.String("w", "", "File specifying a Batsim workload in json format")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig.yaml")
	outDir := flag.String("out", "", "path/to/output/dir/prefix")

	flag.Parse()
	if *wlJson == "" || *kubeconfig == "" || *outDir == "" {
		fmt.Fprintf(os.Stderr, "usage:\n\tbatkube-test -w <workload.json> -config <kubeconfig.yaml> -out output/dir/prefix\n")
		os.Exit(1)
	}

	s := newSubmitterForConfig(*kubeconfig)

	// Parse workload
	wl := parseFile(*wlJson)
	pods := translateJobsToPods(&wl)

	// Initialize
	quit := make(chan struct{})
	defer close(quit)
	initEventInformer(s, quit)
	csvData := initialState(&wl)
	s.unfinishedJobs = len(csvData) - 1

	// Launch the experience
	wg := sync.WaitGroup{}
	wg.Add(2)
	s.origin = time.Now()
	go func() {
		defer wg.Done()
		defer cleanupResources(s)
		runResourceWatcher(s, csvData)
	}()
	go func() {
		defer wg.Done()
		runPodSubmitter(s, pods)
	}()
	wg.Wait()
	computeRemainingData(csvData)

	writeCsv(csvData, *outDir)
}

func writeCsv(csvData [][]string, outDir string) {
	dir := path.Dir(outDir)
	prefix := path.Base(outDir)
	f, err := os.Create(path.Join(dir, prefix+"_jobs.csv"))
	if err != nil {
		log.Fatal(err)
	}
	log.Infoln("Writing output to", f.Name())
	w := csv.NewWriter(f)
	if err := w.WriteAll(csvData); err != nil {
		log.Fatal(err)
	}
}

func newSubmitterForConfig(kubeconfig string) *submitter {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return &submitter{
		ctx:        context.Background(),
		cs:         cs,
		noMoreJobs: make(chan bool),
		events:     make(chan *v1.Event),
	}
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
func runPodSubmitter(s *submitter, pods []*v1.Pod) {
	verifyJobSubmissionOrder(pods) // pods need to be ordered by submission time

	one := int32(1)
	var zero int32
	for _, pod := range pods {
		offsettedSubTime := s.origin.Add(time.Duration(pod.CreationTimestamp.UnixNano()))
		if time.Now().Before(offsettedSubTime) {
			time.Sleep(offsettedSubTime.Sub(time.Now()))
		}
		pod.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		if _, err := s.cs.BatchV1().Jobs(pod.Namespace).Create(s.ctx,
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
	s.noMoreJobs <- true
}

func initialState(wl *translate.Workload) [][]string {
	// This column order must correspond to the indexes defined at the top of this file
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
		csvData[i+1][jobIdIndex] = job.Id
		csvData[i+1][workloadNameIndex] = "w0"
		csvData[i+1][profileIndex] = job.Profile
		csvData[i+1][requestedNumberOfResourcesIndex] = "1"
		csvData[i+1][requestedTimeIndex] = "0" // Time limit on pods is not implemented in batkube
		csvData[i+1][consumedEnergyIndex] = "-1"
		// TODO : handle pods finishing states
		csvData[i+1][finalStateIndex] = "COMPLETED_SUCCESSFULLY"
		csvData[i+1][successIndex] = "1"

	}
	return csvData
}

func computeRemainingData(csvData [][]string) {
	for _, line := range csvData[1:] {
		startingTime, err := strconv.ParseFloat(line[startingTimeIndex], 64)
		if err != nil {
			log.Fatal(err)
		}
		finishingTime, err := strconv.ParseFloat(line[finishTimeIndex], 64)
		if err != nil {
			log.Fatal(err)
		}
		submissionTime, err := strconv.ParseFloat(line[submissionTimeIndex], 64)
		if err != nil {
			log.Fatal(err)
		}
		waitingTime := startingTime - submissionTime
		executionTime := finishingTime - startingTime

		line[executionTimeIndex] = fmt.Sprintf("%f", executionTime)
		line[waitingTimeIndex] = fmt.Sprintf("%f", waitingTime)
		line[turnaroundTimeIndex] = fmt.Sprintf("%f", executionTime+waitingTime)
		line[stretchIndex] = fmt.Sprintf("%f", (finishingTime-submissionTime)/(finishingTime-startingTime))
	}
}

/*
Continuously watches the cluster state and writes the events to csvData
*/
func runResourceWatcher(s *submitter, csvData [][]string) {
	var noMoreJobsBool bool
	for s.unfinishedJobs > 0 || !noMoreJobsBool {
		select {
		case <-s.noMoreJobs:
			noMoreJobsBool = true
		case e := <-s.events:
			log.Infoln(e.Reason, e.InvolvedObject.Kind, e.InvolvedObject.Name)
			handleEvent(s, csvData, e)
		}
	}
}

func handleEvent(s *submitter, csvData [][]string, event *v1.Event) {
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
	// A slight overhead is then added to these times.
	switch event.Reason {
	case "Completed":
		s.unfinishedJobs--
		jobLine[finishTimeIndex] = timeToBatsimTime(time.Now(), s.origin)
	case "SuccessfulCreate":
		jobLine[submissionTimeIndex] = timeToBatsimTime(time.Now(), s.origin)
	case "Scheduled":
		pod, err := s.cs.CoreV1().Pods(event.Namespace).Get(s.ctx, event.InvolvedObject.Name, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		jobLine[scheduledIndex] = timeToBatsimTime(time.Now(), s.origin)
		jobLine[allocatedResourcesIndex] = pod.Spec.NodeName
	case "Pulling":
		jobLine[pullingIndex] = timeToBatsimTime(time.Now(), s.origin)
	case "Pulled":
		jobLine[pulledIndex] = timeToBatsimTime(time.Now(), s.origin)
	case "Started":
		jobLine[startingTimeIndex] = timeToBatsimTime(time.Now(), s.origin)
	case "Created":
		jobLine[createdIndex] = timeToBatsimTime(time.Now(), s.origin)
	default:
	}
}

func timeToBatsimTime(t time.Time, origin time.Time) string {
	return fmt.Sprintf("%f", float64(t.Sub(origin).Round(time.Millisecond))/1e9)
}

func initEventInformer(s *submitter, quit chan struct{}) {
	factory := informers.NewSharedInformerFactory(s.cs, 0)
	factory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.events <- obj.(*v1.Event)
		},
	})
	factory.Start(quit)
}

/*
Cleans up the cluster resources in preparation for the next epoch
*/
func cleanupResources(s *submitter) {
	namespaces, err := s.cs.CoreV1().Namespaces().List(s.ctx, metav1.ListOptions{})
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
		if err := s.cs.BatchV1().Jobs(namespace.Name).DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("jobs cleaned for namespace %s", namespace.Name)
		}
		if err := s.cs.CoreV1().Pods(namespace.Name).DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("pods cleaned for namespace %s", namespace.Name)
		}
		if err := s.cs.CoreV1().Events(namespace.Name).DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}); err != nil {
			log.Warn(err)
		} else {
			log.Infof("events cleaned for namespace %s", namespace.Name)
		}
	}
	log.Info("Done cleaning resources")
}
