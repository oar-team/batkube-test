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
	"github.com/oar-team/batkube/pkg/translate"
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
	nodesId        map[string]int
	jobCompletion  chan string
	epoch          int
}

type csvStruct struct {
	data [][]string
	lock sync.Mutex
}

func main() {
	wlJson := flag.String("w", "", "File specifying a Batsim workload in json format")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig.yaml")
	outDir := flag.String("out", "", "path/to/output/dir/prefix")
	epochs := flag.String("epochs", "", "Number of iterations to run")
	loglevel := flag.String("loglevel", "", "Log level")

	flag.Parse()
	if *wlJson == "" || *kubeconfig == "" || *outDir == "" || *epochs == "" {
		flag.Usage()
		os.Exit(0)
	}

	level, err := log.ParseLevel(*loglevel)
	if err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	s := newSubmitterForConfig(*kubeconfig)
	csv := csvStruct{}

	// Parse workload
	wl := parseFile(*wlJson)
	pods := translateJobsToPods(&wl)

	// Initialize
	quit := make(chan struct{})
	defer close(quit)
	s.initEventInformer(quit)

	// Launch the experience
	epochsValue, err := strconv.Atoi(*epochs)
	check(err)
	s.cleanupResources()
	for s.epoch = 0; s.epoch < epochsValue; s.epoch++ {
		log.Infof("\n========EPOCH %d========\n", s.epoch)
		csv.data = initialState(&wl)
		s.unfinishedJobs = len(csv.data) - 1

		wg := sync.WaitGroup{}
		wg.Add(2)
		s.origin = time.Now()
		go func() {
			defer wg.Done()
			s.runResourceWatcher(&csv)
		}()
		go func() {
			defer wg.Done()
			s.runPodSubmitter(&csv, pods)
		}()
		wg.Wait()
		computeRemainingData(csv.data)

		log.Infof("Epoch done in %s (%d jobs)\n", time.Now().Sub(s.origin), len(csv.data)-1)

		writeCsv(csv.data, *outDir, s.epoch)
		s.cleanupResources()
	}
}

func writeCsv(csvData [][]string, outDir string, epoch int) {
	dir := path.Dir(outDir)
	prefix := path.Base(outDir)
	f, err := os.Create(path.Join(dir, prefix+fmt.Sprintf("%d", epoch)+"_jobs.csv"))
	check(err)
	log.Infoln("Writing output to", f.Name())
	w := csv.NewWriter(f)
	if err := w.WriteAll(csvData); err != nil {
		log.Fatal(err)
	}
}

func newSubmitterForConfig(kubeconfig string) *submitter {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	check(err)
	cs, err := kubernetes.NewForConfig(config)
	check(err)

	return &submitter{
		ctx:           context.Background(),
		cs:            cs,
		noMoreJobs:    make(chan bool),
		events:        make(chan *v1.Event),
		nodesId:       make(map[string]int, 0),
		jobCompletion: make(chan string, 0),
	}
}

/*
Parses a workload file into a byte array
*/
func parseFile(file string) translate.Workload {
	wlFile, err := os.Open(file)
	check(err)
	decoder := json.NewDecoder(wlFile)

	// First step : decode into a map
	jsonData := make(map[string]interface{}, 0)
	err = decoder.Decode(&jsonData)
	check(err)

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
		check(err)

		// models.IoK8sAPICoreV1Pod -> v1.Pod
		// Translation is done thanks to the json tags that remain the same.
		corev1Pod := v1.Pod{}
		podBytes, err := json.Marshal(pod)
		check(err)
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
func (s *submitter) runPodSubmitter(csv *csvStruct, pods []*v1.Pod) {
	verifyJobSubmissionOrder(pods) // pods need to be ordered by submission time
	wg := sync.WaitGroup{}

	one := int32(1)
	var zero int32
	for _, pod := range pods {
		offsettedSubTime := s.origin.Add(time.Duration(pod.CreationTimestamp.UnixNano()))
		if time.Now().Before(offsettedSubTime) {
			time.Sleep(offsettedSubTime.Sub(time.Now()))
		}
		go func(pod *v1.Pod) {
			wg.Add(1)
			defer wg.Done()
			// Job names have format "workload-epoch-jobId"
			pod.Spec.RestartPolicy = v1.RestartPolicyOnFailure
			jobName := fmt.Sprintf("%s%d-%s", pod.Name[:3], s.epoch, pod.Name[3:])
			_, err := s.cs.BatchV1().Jobs(pod.Namespace).Create(s.ctx,
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name: jobName,
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
			)
			check(err)
			csv.write(csv.getLine(strings.Split(jobName, "-")[2]), submissionTimeIndex, timeToBatsimTime(time.Now(), s.origin))
			log.Infof("job %s submitted\n", jobName)
		}(pod)
	}
	wg.Wait()
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
		csvData[i+1][submissionTimeIndex] = fmt.Sprintf("%f", job.Subtime) // job subtime is already formatted in seconds
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
		check(err)
		finishingTime, err := strconv.ParseFloat(line[finishTimeIndex], 64)
		check(err)
		submissionTime, err := strconv.ParseFloat(line[submissionTimeIndex], 64)
		check(err)
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
func (s *submitter) runResourceWatcher(csv *csvStruct) {
	var noMoreJobsBool bool
	for s.unfinishedJobs > 0 || !noMoreJobsBool {
		if s.unfinishedJobs < 0 {
			log.Fatal("Numer of unfinishedJobs is negative")
		}
		select {
		case <-s.noMoreJobs:
			noMoreJobsBool = true
		case e := <-s.events:
			s.handleEvent(csv, e)
		case jobName := <-s.jobCompletion:
			// This code duplicates on handleEvent. This is a
			// hotfix to the lack of 'Completed' events issue.
			sp := strings.Split(jobName, "-")
			jobEpoch, err := strconv.Atoi(sp[1])
			check(err)
			if jobEpoch != s.epoch {
				// There is an issue with this channel receiving
				// updates from last epoch
				continue
			}
			id := sp[2]
			lineIndex := csv.getLine(id)
			s.unfinishedJobs--
			log.Infof("Job %s completed (%d jobs remaining)\n", jobName, s.unfinishedJobs)
			csv.write(lineIndex, finishTimeIndex, timeToBatsimTime(time.Now(), s.origin))
		}
	}
}

func (s *submitter) handleEvent(csv *csvStruct, event *v1.Event) {
	podNameSplt := strings.Split(event.InvolvedObject.Name, "-")
	if len(podNameSplt) == 0 || podNameSplt[0] != "w0" {
		return
	}

	lineIndex := csv.getLine(podNameSplt[2])
	// Some events are not related to jobs or pods
	if lineIndex == 0 {
		return
	}
	log.Debugln(event.Reason, event.InvolvedObject.Kind, event.InvolvedObject.Name)

	switch event.Reason {
	case "Scheduled":
		pod, err := s.cs.CoreV1().Pods(event.Namespace).Get(s.ctx, event.InvolvedObject.Name, metav1.GetOptions{})
		check(err)
		csv.write(lineIndex, scheduledIndex, timeToBatsimTime(time.Now(), s.origin))
		nodeId, ok := s.nodesId[pod.Spec.NodeName]
		if !ok {
			n := len(s.nodesId)
			s.nodesId[pod.Spec.NodeName] = n
			nodeId = n
		}
		csv.write(lineIndex, allocatedResourcesIndex, fmt.Sprintf("%d", nodeId))
	case "Pulling":
		csv.write(lineIndex, pullingIndex, timeToBatsimTime(time.Now(), s.origin))
	case "Pulled":
		csv.write(lineIndex, pulledIndex, timeToBatsimTime(time.Now(), s.origin))
	case "Started":
		csv.write(lineIndex, startingTimeIndex, timeToBatsimTime(time.Now(), s.origin))
	case "Created":
		csv.write(lineIndex, createdIndex, timeToBatsimTime(time.Now(), s.origin))
	default:
	}
}

func timeToBatsimTime(t time.Time, origin time.Time) string {
	return fmt.Sprintf("%f", float64(t.Sub(origin).Round(time.Millisecond))/1e9)
}

func (s *submitter) initEventInformer(quit chan struct{}) {
	factory := informers.NewSharedInformerFactory(s.cs, 0)
	factory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.events <- obj.(*v1.Event)
		},
	})
	factory.Batch().V1().Jobs().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Completions events are no longer sent so we need to
			// find another way
			job := newObj.(*batchv1.Job)
			if job.Status.Succeeded == 1 {
				s.jobCompletion <- job.Name
			}
		},
	})
	factory.Start(quit)
}

/*
Cleans up the cluster resources in preparation for the next epoch
*/
func (s *submitter) cleanupResources() {
	var zero int64
	log.Infoln("Waiting a bit for resources to stabilize before cleaning...")
	time.Sleep(5 * time.Second)

	log.Infoln("cleaning jobs...")
	check(s.cs.BatchV1().Jobs("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
	time.Sleep(1 * time.Second)
	jobList, err := s.cs.BatchV1().Jobs("default").List(s.ctx, metav1.ListOptions{})
	for err == nil && len(jobList.Items) > 0 {
		check(s.cs.BatchV1().Jobs("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
		time.Sleep(1 * time.Second)
		jobList, err = s.cs.BatchV1().Jobs("default").List(s.ctx, metav1.ListOptions{})
	}
	check(err)

	log.Infoln("cleaning pods...")
	check(s.cs.CoreV1().Pods("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
	time.Sleep(1 * time.Second)
	podList, err := s.cs.CoreV1().Pods("default").List(s.ctx, metav1.ListOptions{})
	for err == nil && len(podList.Items) > 0 {
		check(s.cs.CoreV1().Pods("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
		time.Sleep(1 * time.Second)
		podList, err = s.cs.CoreV1().Pods("default").List(s.ctx, metav1.ListOptions{})
	}
	check(err)

	log.Infoln("cleaning core v1 events...")
	check(s.cs.CoreV1().Events("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
	time.Sleep(1 * time.Second)
	coreEventList, err := s.cs.CoreV1().Events("default").List(s.ctx, metav1.ListOptions{})
	for err == nil && len(coreEventList.Items) > 0 {
		check(s.cs.CoreV1().Events("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
		time.Sleep(1 * time.Second)
		coreEventList, err = s.cs.CoreV1().Events("default").List(s.ctx, metav1.ListOptions{})
	}
	check(err)

	log.Infoln("cleaning events v1 beta 1 events...")
	check(s.cs.CoreV1().Events("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
	time.Sleep(1 * time.Second)
	eventList, err := s.cs.EventsV1beta1().Events("default").List(s.ctx, metav1.ListOptions{})
	for err == nil && len(eventList.Items) > 0 {
		check(s.cs.CoreV1().Events("default").DeleteCollection(s.ctx, metav1.DeleteOptions{GracePeriodSeconds: &zero}, metav1.ListOptions{}))
		time.Sleep(1 * time.Second)
		eventList, err = s.cs.EventsV1beta1().Events("default").List(s.ctx, metav1.ListOptions{})
	}
	check(err)

	log.Info("Done cleaning resources")
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (csv *csvStruct) write(line, col int, input string) {
	csv.lock.Lock()
	defer csv.lock.Unlock()
	csv.data[line][col] = input
}

func (csv *csvStruct) getLine(id string) int {
	csv.lock.Lock()
	defer csv.lock.Unlock()
	for i, line := range csv.data {
		if line[0] == id {
			return i
		}
	}
	return 0
}
