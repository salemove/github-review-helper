package main

import (
	"log"
	"net/http"
)

type Handler func(http.ResponseWriter, *http.Request) Response

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := h(w, r)
	log.Println("Responding to the HTTP request with:")
	response.logResponse()
	response.WriteResponse(w)
}

type Response interface {
	WriteResponse(http.ResponseWriter)
	logResponse()
}

type ErrorResponse struct {
	Error        error
	Code         int
	ErrorMessage string
}

func (r ErrorResponse) WriteResponse(w http.ResponseWriter) {
	http.Error(w, r.ErrorMessage, r.Code)
}

func (r ErrorResponse) logResponse() {
	if r.Error != nil {
		log.Printf("Error: %s: %v\n", r.ErrorMessage, r.Error)
	} else {
		log.Printf("Error: %s\n", r.ErrorMessage)
	}
}

type SuccessResponse struct {
	Message string
}

func (r SuccessResponse) WriteResponse(w http.ResponseWriter) {
	w.Write([]byte(r.Message))
}

func (r SuccessResponse) logResponse() {
	log.Printf("Success: %s\n", r.Message)
}

// handleAsyncResponse provides consistent error/success logging for operations
// that are left to continue working after the original HTTP request that
// initiated the operation has been handled and closed.
func handleAsyncResponse(response Response) {
	log.Println("Finishing an asynchronous operation with:")
	response.logResponse()
}
