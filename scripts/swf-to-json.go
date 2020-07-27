package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
)

func main() {
	filePath := flag.String("in", "", "input csv file with swf format")
	outPath := flag.String("out", "", "output file")
	flag.Parse()
	if *filePath == "" || *outPath == "" {
		flag.Usage()
		os.Exit(1)
	}

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
		line := parseLine(scanner.Text())
		if line == nil {
			continue
		}

		// Extract the necessary info
		subtime, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			panic(err)
		}
		runTime, err := strconv.ParseFloat(line[3], 64)
		if err != nil {
			panic(err)
		}
		cpu := line[4]

		// Create the profile if it does not exist
		profileName := fmt.Sprintf("delay%f", runTime)
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

	if err = scanner.Err(); err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(out)
	encodeWorkload(&wl, encoder)
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
			"ret":       prof.Ret,
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

func parseLine(line string) []string {
	if len(line) == 0 || line[0] == ';' {
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
