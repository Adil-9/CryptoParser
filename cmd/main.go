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
	"time"

	"gopkg.in/yaml.v3"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "f", "", "provide config file path") //getting path to config file
	flag.Parse()
	if filePath == "" {
		log.Fatal("Config file path not provided")
	}

	var data configuration.Configurations //config file parser struct
	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = yaml.Unmarshal(file, &data) //parsing config file
	if err != nil {
		log.Fatal(err.Error())
	} else if data.MaxWorkers <= 0 {
		log.Fatal("max_workers set to 0, can not run program with such value")
	} else if data.MaxWorkers > runtime.NumCPU() { //setting max cores if maxWorkers exceeds
		data.MaxWorkers = runtime.NumCPU()
	}

	var handlersList []*runner.Handler
	wg := sync.WaitGroup{}

	stopChannel := make(chan struct{}) //channel to indicate graceful shutdown

	output := make(chan string, 1024) //buffered printing channel
	go printer(output, &wg)           //starting a go routine for printer

	for i := 1; i <= data.MaxWorkers; i++ { //initiating handlers and distributing cryptopairs
		hand := runner.Handler{}
		handlersList = append(handlersList, &hand)
		hand.Stop = stopChannel
		hand.OutChannel = output
		for j := i - 1; j < len(data.Symbols); j += data.MaxWorkers {
			coin := runner.Coins{Symbol: data.Symbols[j]}
			hand.Coins = append(hand.Coins, coin)
		}
		wg.Add(1)
		go hand.Run(&wg)
	}

	go requestCounter(handlersList) //func to print number of requests every 5 seconds

	reader := bufio.NewReader(os.Stdin)
	for { //reading "STOP" to stop program
		stop, _ := reader.ReadString('\n')
		stop = strings.TrimSpace(stop)
		if stop == "STOP" {
			close(stopChannel) //indicating to routines that program should be stopped (for graceful shutdown)
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
		fmt.Print(text)
	}
}

func requestCounter(hands []*runner.Handler) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var count int
		for i := range hands {
			count += hands[i].GetRequestsCount()
		}
		fmt.Printf("workers requests total: %d\n", count)
	}
}
