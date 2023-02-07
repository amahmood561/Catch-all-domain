package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type DomainCounter struct {
	Delivered int
	Bounced   int
	Status    string
}

var domainCounts map[string]*DomainCounter

func main() {
	// Initialize the domainCounts map
	domainCounts = make(map[string]*DomainCounter)

	// Create a new router
	router := mux.NewRouter()

	// Handle PUT requests for delivered events
	router.HandleFunc("/events/{domain}/delivered", HandleDeliveredEvent).Methods("PUT")

	// Handle PUT requests for bounced events
	router.HandleFunc("/events/{domain}/bounced", HandleBouncedEvent).Methods("PUT")

	// Handle GET requests for domain statuses
	router.HandleFunc("/domains/{domain}", HandleGetDomainStatus).Methods("GET")

	http.ListenAndServe(":8000", router)
}
func HandleDeliveredEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Initialize the domain counter if it doesn't exist
	if _, ok := domainCounts[domain]; !ok {
		domainCounts[domain] = &DomainCounter{
			Delivered: 0,
			Bounced:   0,
			Status:    "unknown",
		}
	}

	// Increment the delivered count
	domainCounts[domain].Delivered++

	// Check if the domain is a catch-all
	if domainCounts[domain].Delivered >= 1000 && domainCounts[domain].Bounced == 0 {
		domainCounts[domain].Status = "catch-all"
	}

	w.WriteHeader(http.StatusOK)
}

func HandleBouncedEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Initialize the domain counter if it doesn't exist
	if _, ok := domainCounts[domain]; !ok {
		domainCounts[domain] = &DomainCounter{
			Delivered: 0,
			Bounced:   0,
			Status:    "unknown",
		}
	}

	// Increment the bounced count
	domainCounts[domain].Bounced++

	// Update the status to not catch-all
	domainCounts[domain].Status = "not catch-all"

	w.WriteHeader(http.StatusOK)
}

func HandleGetDomainStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Check if the domain counter exists
	if _, ok := domainCounts[domain]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get the status of the domain
	status := domainCounts[domain].Status

	// Prepare the response
	response := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	// Write the response as JSON
	json.NewEncoder(w).Encode(response)
}
