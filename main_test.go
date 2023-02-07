package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDeliveredEvent(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()
	domainCounts = make(map[string]*DomainCounter)

	// Register the handleDeliveredEvent function with the router
	router.HandleFunc("/events/{domain}/delivered", HandleDeliveredEvent).Methods("PUT")

	// Create a new request
	req, _ := http.NewRequest("PUT", "/events/example.com/delivered", nil)

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Pass the request and response recorder to the router
	router.ServeHTTP(rr, req)

	// Assert that the response has a status code of 200 OK
	assert.Equal(t, http.StatusOK, rr.Code)

	// Assert that the domain counter for example.com has been incremented
	assert.Equal(t, 1, domainCounts["example.com"].Delivered)
}

func TestHandleDeliveredEvent_CatchAll(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()
	domainCounts = make(map[string]*DomainCounter)

	// Register the handleDeliveredEvent function with the router
	router.HandleFunc("/events/{domain}/delivered", HandleDeliveredEvent).Methods("PUT")

	// Initialize the domain's delivered and bounced count and status
	domainCounts["example.com"] = &DomainCounter{
		Delivered: 999,
		Bounced:   0,
		Status:    "unknown",
	}

	// Create a new request
	req, err := http.NewRequest("PUT", "/events/example.com/delivered", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Send the request and response through the router
	router.ServeHTTP(rr, req)

	// Assert that the response status code is 200 OK
	assert.Equal(t, http.StatusOK, rr.Code)

	// Assert that the delivered count for
	// Assert that the domain counter for example.com has been incremented
	assert.Equal(t, 1000, domainCounts["example.com"].Delivered)
}

func TestHandleBouncedEvent(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()
	domainCounts = make(map[string]*DomainCounter)

	// Register the handleBouncedEvent function with the router
	router.HandleFunc("/events/{domain}/bounced", HandleBouncedEvent).Methods("PUT")

	// Create a new request
	req, err := http.NewRequest("PUT", "/events/example.com/bounced", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Send the request and response through the router
	router.ServeHTTP(rr, req)

	// Assert that the response status code is 200 OK
	assert.Equal(t, http.StatusOK, rr.Code)

	// Assert that the bounced count for the domain is 1
	assert.Equal(t, 1, domainCounts["example.com"].Bounced)

	// Assert that the status of the domain is "not catch-all"
	assert.Equal(t, "not catch-all", domainCounts["example.com"].Status)
}

func TestHandleGetDomainStatus(t *testing.T) {
	router := mux.NewRouter()
	// setup
	domainCounts = make(map[string]*DomainCounter)
	domainCounts["example.com"] = &DomainCounter{
		Delivered: 1500,
		Bounced: 0,
		Status: "catch-all",
	}
	req, _ := http.NewRequest("GET", "/domains/example.com", nil)

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(HandleGetDomainStatus)
	router.HandleFunc("/domains/{domain}", HandleGetDomainStatus).Methods("GET")

	// test
	router.ServeHTTP(rr, req)

	// check status code
	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200")

	// check response body
	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "catch-all", response["status"], "status should be catch-all")
}
