package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube/pkg/translate"
)

func main() {

	wlJson := flag.String("w", "", "File specifying a Batsim workload in json format")

	flag.Parse()
	if flag.NFlag() == 0 {
		fmt.Fprintf(os.Stderr, "usage:\n\tbatkube-test -w <workload.json> -config <kubeconfig.yaml>\n")
		os.Exit(1)
	}

	wl := parseFile(*wlJson)

	fmt.Println(wl)
}

/*
Parses a workload file into a byte array
*/
func parseFile(file string) translate.Workload {
	wlFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(wlFile)
	wlStruct := translate.Workload{}
	err = decoder.Decode(&wlStruct)
	if err != nil {
		panic(err)
	}
	return wlStruct
}
