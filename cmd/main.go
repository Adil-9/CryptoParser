package main

import (
	"CryptoParser/internal/configuration"
	"CryptoParser/internal/runner"
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

func main() {
	var filePath string

	flag.StringVar(&filePath, "f", "", "provide config file") //getting path to config file
	flag.Parse()
	if filePath == "" {
		log.Fatal("Config file not provided")
	}

	var data configuration.Configurations

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = yaml.Unmarshal(file, &data) //parsing config file
	if err != nil {
		log.Fatal(err.Error())
	} else if data.MaxWorkers == 0 {
		log.Fatal("Can not work with 0 cores")
	}

	if data.MaxWorkers > runtime.NumCPU() { //setting max cores if maxWorkers exceeds
		data.MaxWorkers = runtime.NumCPU()
	}

	wg := sync.WaitGroup{}

	output := make(chan string, 1024) //printing channel
	go printer(output, &wg)

	StopChannel := make(chan struct{})
	for i := 1; i <= data.MaxWorkers; i++ {
		hand := runner.Handler{}
		hand.Stop = StopChannel
		hand.OutChannel = output
		for j := i - 1; j < len(data.Symbols); j += data.MaxWorkers {
			hand.AppendCoinsSymbol(data.Symbols[j])
		}
		wg.Add(1)
		go hand.Run(&wg)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		stop, _ := reader.ReadString('\n')
		stop = strings.TrimSpace(stop)
		if stop == "STOP" {
			close(StopChannel)
			break
		}
	}
	wg.Wait() //waiting for all goroutines to stop
	wg.Add(1)
	close(output)
	wg.Wait() //waiting for printer to print
}

func printer(str <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for text := range str {
		fmt.Println(text)
	}
}
