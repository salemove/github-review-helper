package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

func delay(duration time.Duration, operation func() Response, asyncOperationWg *sync.WaitGroup) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	timer := time.NewTimer(duration)

	asyncOperationWg.Add(1)
	go func() {
		defer asyncOperationWg.Done()
		// Avoid leaking channels
		defer signal.Stop(interruptChan)

		// Block until either of the 2 channels receives.
		select {
		case <-interruptChan:
			log.Println("Received an interrupt signal (SIGINT). Starting a scheduled process immediately.")
		case <-timer.C:
		}

		response := operation()
		handleAsyncResponse(response)
	}()
}
