package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

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

	out, err := os.Create(*outPath)
	if err != nil {
		panic(err)
	}

	r := csv.NewReader(f)
	r.Comment = ';'
	r.Comma = ' '

	wl := translate.Workload{
		NbRes:    1,
		Jobs:     make([]translate.Job, 0),
		Profiles: make(map[string]translate.Profile),
	}
	line, _ := r.Read()
	for line != nil {
		// There are many blank spaces
		line = parseLine(line)

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

		line, _ = r.Read()
	}

	encoder := json.NewEncoder(out)
	if err = encoder.Encode(wl); err != nil {
		panic(err)
	}
}

func parseLine(line []string) []string {
	formattedLine := make([]string, 0)
	for _, col := range line {
		if col != "" {
			formattedLine = append(formattedLine, col)
		}
	}
	return formattedLine
}
