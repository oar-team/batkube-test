package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
)

var maxCPU float64 // to normalize cpu usage

func main() {
	filePath := flag.String("in", "", "input csv file with swf format")
	outPath := flag.String("out", "", "output file")
	normalize := flag.Float64("norm", 0, "normalize cpu usage between 0 and input value")
	uniform := flag.Float64("uniform", 0, "uniformize cpu usage to given value")
	trim := flag.Duration("trim", 0, "trim job durations to keep them under this value")
	flag.Parse()
	if *filePath == "" || *outPath == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *normalize != 0 && *uniform != 0 {
		fmt.Println("can't set both uniformize and normalize")
		flag.Usage()
		os.Exit(1)
	}

	trimFloat := float64(*trim) / 1e9

	f, err := os.Open(*filePath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	out, err := os.Create(*outPath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)

	wl := translate.Workload{
		NbRes:    1,
		Jobs:     make([]translate.Job, 0),
		Profiles: make(map[string]translate.Profile),
	}

	for scanner.Scan() {
		parseLine(scanner.Text(), &wl)
	}
	if err = scanner.Err(); err != nil {
		panic(err)
	}

	// offsets all jobs subtimes so the first job corresponds to the origin
	if len(wl.Jobs) == 0 {
		panic(errors.New("This workload has no jobs"))
	}
	offset := wl.Jobs[0].Subtime
	for i := range wl.Jobs {
		wl.Jobs[i].Subtime -= offset
	}

	for _, prof := range wl.Profiles {
		cpu := prof.Specs["cpu"].(float64)
		if cpu > maxCPU {
			maxCPU = cpu
		}
	}

	// Normalize cpu usage
	for _, prof := range wl.Profiles {
		if *normalize > 0 {
			prof.Specs["cpu"] = *normalize * prof.Specs["cpu"].(float64) / maxCPU
		} else if *uniform > 0 {
			prof.Specs["cpu"] = *uniform
		}
		if prof.Specs["cpu"].(float64) < 0.001 {
			// resources requests can not be lower than 1m
			prof.Specs["cpu"] = float64(0.001)
		}
		cpu := prof.Specs["cpu"].(float64)
		prof.Specs["cpu"] = math.Round(cpu*1000) / 1000
		if *trim > 0 && prof.Specs["delay"].(float64) > trimFloat {
			prof.Specs["delay"] = trimFloat
		}
	}

	encoder := json.NewEncoder(out)
	encodeWorkload(&wl, encoder)
}

func parseLine(lineStr string, wl *translate.Workload) {
	line := parseLineStringToSlice(lineStr)
	if line == nil {
		return
	}

	// Extract the necessary info
	runTime, err := strconv.ParseFloat(line[3], 64)
	if err != nil {
		panic(err)
	}
	if runTime == float64(0) {
		return
	}

	subtime, err := strconv.ParseFloat(line[1], 64)
	if err != nil {
		panic(err)
	}
	cpu, err := strconv.ParseFloat(line[4], 64)
	if err != nil {
		panic(err)
	}

	// Create the profile if it does not exist
	profileName := fmt.Sprintf("delay%d", int64(runTime))
	_, ok := wl.Profiles[profileName]
	if !ok {
		wl.Profiles[profileName] = translate.Profile{
			Type: "delay",
			Ret:  1,
			Specs: map[string]interface{}{
				"scheduler": "default",
				"delay":     runTime,
				"cpu":       cpu,
			},
		}
	}

	job := translate.Job{
		Id:      line[0],
		Subtime: subtime,
		Res:     1,
		Profile: profileName,
	}
	wl.Jobs = append(wl.Jobs, job)
}

/*
Workaround to the profile spec encoding problem.
Works ONLY for workload consisting of delay profiles
*/
func encodeWorkload(wl *translate.Workload, e *json.Encoder) {
	wlMap := map[string]interface{}{
		"nb_res":   wl.NbRes,
		"jobs":     wl.Jobs,
		"profiles": make(map[string]interface{}, 0),
	}
	for profName, prof := range wl.Profiles {
		profMap := map[string]interface{}{
			"type":      prof.Type,
			"delay":     prof.Specs["delay"],
			"scheduler": prof.Specs["scheduler"],
			"cpu":       prof.Specs["cpu"],
		}
		wlMap["profiles"].(map[string]interface{})[profName] = profMap
	}
	if err := e.Encode(wlMap); err != nil {
		panic(err)
	}
}

func parseLineStringToSlice(line string) []string {
	//var err error
	if len(line) == 0 || line[0] == ';' {
		//if strings.Contains(line, "MaxProcs") {
		//	maxCPU, err = strconv.ParseFloat(strings.Split(line, ": ")[1], 64)
		//	if err != nil {
		//		panic(err)
		//	}
		//}
		return nil
	}

	formattedLine := make([]string, 0)
	line = strings.ReplaceAll(line, "\t", " ")
	split := strings.Split(line, " ")
	for _, col := range split {
		if col != "" {
			formattedLine = append(formattedLine, col)
		}
	}
	return formattedLine
}
