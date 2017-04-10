package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type asyncResponse struct {
	Response
	MayBeRetried bool
}

type asyncErrorResponse struct {
	ErrorResponse
	MayBeRetried bool
}

// MaybeSyncResponse can be returned from operations which may or may not
// complete synchronously. When OperationFinishedSynchronously is true, then
// Response will be specified.
type MaybeSyncResponse struct {
	Response
	OperationFinishedSynchronously bool
}

func syncResponse(response Response) MaybeSyncResponse {
	return MaybeSyncResponse{
		Response:                       response,
		OperationFinishedSynchronously: true,
	}
}

func delayWithRetries(tryDelays []time.Duration, operation func() asyncResponse,
	asyncOperationWg *sync.WaitGroup) MaybeSyncResponse {

	if len(tryDelays) < 1 {
		return syncResponse(ErrorResponse{
			Code:         http.StatusInternalServerError,
			ErrorMessage: "Cannot schedule any delayed operations when tryDelays is empty",
		})
	}

	if tryDelays[0] == 0 {
		response := operation()
		if len(tryDelays) > 1 && response.MayBeRetried {
			log.Println("Operation will be retried")
			if err := asyncDelayWithRetries(tryDelays[1:], operation, asyncOperationWg); err != nil {
				return syncResponse(
					ErrorResponse{err, http.StatusInternalServerError, "Failed to schedule async retries"},
				)
			}
			return MaybeSyncResponse{OperationFinishedSynchronously: false}
		}
		return syncResponse(response)
	}

	if err := asyncDelayWithRetries(tryDelays, operation, asyncOperationWg); err != nil {
		return syncResponse(
			ErrorResponse{err, http.StatusInternalServerError, "Failed to schedule async delay with retries"},
		)
	}
	return MaybeSyncResponse{OperationFinishedSynchronously: false}
}

func asyncDelayWithRetries(tryDelays []time.Duration, operation func() asyncResponse,
	asyncOperationWg *sync.WaitGroup) error {

	if len(tryDelays) < 1 {
		return errors.New("Cannot schedule any delayed operations when tryDelays is empty")
	}

	delay(tryDelays[0], func() {
		response := operation()
		handleAsyncResponse(response.Response)
		if len(tryDelays) > 1 && response.MayBeRetried {
			log.Println("Operation will be retried")
			if err := asyncDelayWithRetries(tryDelays[1:], operation, asyncOperationWg); err != nil {
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

func retriable(response Response) asyncResponse {
	return asyncResponse{
		Response:     response,
		MayBeRetried: true,
	}
}

func nonRetriable(response Response) asyncResponse {
	return asyncResponse{
		Response:     response,
		MayBeRetried: false,
	}
}

func retriableError(errResp ErrorResponse) *asyncErrorResponse {
	return &asyncErrorResponse{
		ErrorResponse: errResp,
		MayBeRetried:  true,
	}
}

func nonRetriableError(errResp ErrorResponse) *asyncErrorResponse {
	return &asyncErrorResponse{
		ErrorResponse: errResp,
		MayBeRetried:  false,
	}
}

func (a asyncErrorResponse) toAsyncResponse() asyncResponse {
	return asyncResponse{
		Response:     a.ErrorResponse,
		MayBeRetried: a.MayBeRetried,
	}
}
