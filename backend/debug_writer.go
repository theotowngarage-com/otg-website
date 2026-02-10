package main

import (
	"bytes"
	"net/http"
)

// CustomResponseWriter captures the status code and response body
// Initialise:
//
//	crw := &CustomResponseWriter{
//		ResponseWriter: w,
//		body:           &bytes.Buffer{},
//		headers:        make(http.Header),
//		statusCode:     http.StatusOK,
//	}
type CustomResponseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	headers    http.Header
	statusCode int
}

// Implement Write method
func (w *CustomResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // Capture response
	return w.ResponseWriter.Write(b)
}

// Implement WriteHeader method
//
// // Run actual handler
// actualHandler(crw, r)
//
// // Logging after handler
// fmt.Println("---- Response Info ----")
// fmt.Println("Status Code:", crw.statusCode)
// fmt.Println("Headers:")
//
//	for k, v := range crw.headers {
//	    fmt.Printf("  %s: %s\n", k, v)
//	}
//
// fmt.Println("Body:", crw.body.String())
// fmt.Println("-----------------------")
func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// Copy headers for logging
	for k, v := range w.ResponseWriter.Header() {
		w.headers[k] = v
	}
	w.ResponseWriter.WriteHeader(statusCode)
}
