package main

import (
	"log"
	"net/http"
)

type Handler func(http.ResponseWriter, *http.Request) Response

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := h(w, r)
	response.WriteResponse(w)
}

type Response interface {
	WriteResponse(http.ResponseWriter)
}

type ErrorResponse struct {
	Error        error
	Code         int
	ErrorMessage string
}

func (r ErrorResponse) WriteResponse(w http.ResponseWriter) {
	if r.Error != nil {
		log.Printf("%s: %v\n", r.ErrorMessage, r.Error)
	} else {
		log.Println(r.ErrorMessage)
	}
	http.Error(w, r.ErrorMessage, r.Code)
}

type SuccessResponse struct {
	Message string
}

func (r SuccessResponse) WriteResponse(w http.ResponseWriter) {
	log.Printf("Responding with a success message: %s\n", r.Message)
	w.Write([]byte(r.Message))
}
