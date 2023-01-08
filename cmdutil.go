package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type Saved struct {
	Saves []Save `json:"saves"`
}

type Save struct {
	Command   string `json:"Command"`
	MaxRSS    string `json:"Max RSS"`
	Systime   string `json:"Sys Time"`
	Usrtime   string `json:"User Time"`
	Actual    string `json:"Actual Time"`
	Nvcsw     string `json:"Context Switch"`
	Timestamp string `json:"Timestamp"`
}

func main() {
	var inputCMD string
	var hide, save bool

	flag.BoolVar(&hide, "hide", false, "Hide/Discard Command Output")
	flag.BoolVar(&save, "save", false, "Save stats in .cmdutil.json file")
	flag.Parse()

	if os.Getenv("SAVE") != "" {
		save = true
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		fmt.Println("cmdutil only supports linux & darwin through Rusage")
		os.Exit(1)
	}

	inputCMD = strings.Join(flag.Args(), " ")

	cmd := exec.Command("/bin/sh", "-c", inputCMD)
	cmd.Stdin = os.Stdin

	if hide {
		cmd.Stderr = io.Discard
		cmd.Stdout = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	start := time.Now()
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	timetaken := time.Since(start)

	if cmd.ProcessState != nil {
		usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
		if !ok {
			fmt.Println("Something Went Wrong. Failed to Assert Rusage")
		} else {
			fmt.Println()
			for i := 0; i < 30; i++ {
				fmt.Print("-")
			}
			instance := Save{
				Command:   inputCMD,
				MaxRSS:    fmt.Sprint(usage.Maxrss / (1024 * 1024)),
				Systime:   time.Duration(usage.Stime.Usec).String(),
				Usrtime:   time.Duration(usage.Utime.Usec).String(),
				Actual:    timetaken.String(),
				Nvcsw:     fmt.Sprint(usage.Nvcsw),
				Timestamp: time.Now().Format(time.UnixDate),
			}
			fmt.Printf("\nCommand: %v\n", instance.Command)
			fmt.Printf("Max RSS: %v MB\n", instance.MaxRSS)
			fmt.Printf("Sys Time: %v\nUser Time: %v\nActual Time: %v\n", instance.Systime, instance.Usrtime, instance.Actual)
			fmt.Printf("Voluntary Context Switch (nvcsw): %v\n", instance.Nvcsw)

			if save {
				var z Saved
				bin, err := os.ReadFile(".cmdutil.json")
				if err == nil {
					_ = json.Unmarshal(bin, &z)
				}
				if z.Saves == nil {
					z.Saves = []Save{}
				}
				z.Saves = append(z.Saves, instance)

				out, erx := json.MarshalIndent(z, "", "  ")
				if erx == nil {
					os.WriteFile(".cmdutil.json", out, 0644)
				}
			}
		}
	}

}
