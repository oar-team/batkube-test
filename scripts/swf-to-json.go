package main

import (
	"encoding/csv"
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	filePath := flag.String("in", "", "input csv file with swf format")
	flag.Parse()
	if *filePath == "" {
		flag.Usage()
		os.Exit(1)
	}
	f, err := os.Open(*filePath)
	if err != nil {
		log.Fatal(err)
	}

	r := csv.NewReader(f)
	r.Comma = ' '
	r.Comment = ';'

	line, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	line = formatLine(line)
}

func formatLine(line []string) []string {
	formattedLine := make([]string, 0)
	for _, col := range line {
		if col != "" {
			formattedLine = append(formattedLine, col)
		}
	}
	return formattedLine
}
