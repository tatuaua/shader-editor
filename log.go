package main

import (
	"fmt"
	"os"
)

var writeLogs bool
var logs []string

func log(format string, args ...any) {
	logs = append(logs, fmt.Sprintf(format, args...))
}

func saveLogs() {
	f, err := os.Create("shader.log")
	if err != nil {
		return
	}
	defer f.Close()
	for _, l := range logs {
		f.WriteString(l + "\n")
	}
}
