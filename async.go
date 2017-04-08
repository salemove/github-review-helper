package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

type asyncResponse struct {
	Response
	MayBeRetried bool
}

func delayWithRetries(tryDelays []time.Duration, operation func() asyncResponse,
	asyncOperationWg *sync.WaitGroup) error {

	if len(tryDelays) < 1 {
		return errors.New("Cannot schedule any delayed operations when tryDelays is empty")
	}

	delay(tryDelays[0], func() {
		response := operation()
		handleAsyncResponse(response.Response)
		if len(tryDelays) > 1 && response.MayBeRetried {
			log.Println("Operation will be retried")
			if err := delayWithRetries(tryDelays[1:], operation, asyncOperationWg); err != nil {
				log.Printf("Failed to schedule another try to start in %s\n", tryDelays[1].String())
				return
			}
		}
	}, asyncOperationWg)
	log.Printf("Scheduled an asynchronous operation to start in %s\n", tryDelays[0].String())
	return nil
}

func delay(duration time.Duration, operation func(), asyncOperationWg *sync.WaitGroup) {
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

		operation()
	}()
}
