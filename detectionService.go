package main

import (
	"context"
	"encoding/json"
	_ "encoding/json"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2"
	"net/http"
	"time"
)

type DomainCounter struct {
	Domain    string
	Delivered int
	Bounced   int
	Status    string
}

const URI_MONGODB = "mongodb://localhost:27017"
const URL_PATH_DELVIERED = "/events/{domain}/delivered"
const URL_PATHS_BOUNCED = "/events/{domain}/bounced"
const URL_PATH_DOMAIN = "/domains/{domain}"
const DataBase = "catch_all"
const Collection = "domain"

func main() {
	// Initialize a new MongoDB client TODO pass client into endpoints and update tests
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		panic(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		panic(err)
	}

	// Create a new router
	router := mux.NewRouter()

	// Handle PUT requests for delivered events
	router.HandleFunc(URL_PATH_DELVIERED, func(w http.ResponseWriter, r *http.Request) {
		HandleDeliveredEvent(w, r)
	}).Methods("PUT")

	// Handle PUT requests for bounced events
	router.HandleFunc(URL_PATHS_BOUNCED, func(w http.ResponseWriter, r *http.Request) {
		HandleBouncedEvent(w, r)
	}).Methods("PUT")

	// Handle GET requests for domain statuses
	router.HandleFunc(URL_PATH_DOMAIN, HandleGetDomainStatus).Methods("GET")
	http.ListenAndServe(":8000", router)

}

// HandleDeliveredEvent
//Description: This method is responsible for handling the PUT requests to the endpoint /events/{domain}/delivered.
//It takes in a request and response writer as input and extracts the domain name from the request.
//It then increments the "delivered" count for the
//domain in the MongoDB collection and updates the status of the domain to "catch-all" if it
//has received more than 1000 "delivered" events and 0 "bounced" events.
///*
func HandleDeliveredEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Validate input
	if domain == "" {
		http.Error(w, "Invalid domain name", http.StatusBadRequest)
		return
	}

	// Connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get a handle to the "domain_counts" collection
	domainCountsCollection := client.Database(DataBase).Collection(Collection)

	// Check if the domain counter exists
	var domainCounter DomainCounter
	err = domainCountsCollection.FindOne(ctx, bson.M{"domain": domain}).Decode(&domainCounter)
	if err == mongo.ErrNoDocuments {
		// Initialize the domain counter if it doesn't exist
		domainCounter = DomainCounter{
			Domain:    domain,
			Delivered: 0,
			Bounced:   0,
			Status:    "unknown",
		}
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Increment the delivered count
	domainCounter.Delivered++

	// Check if the domain is a catch-all
	if domainCounter.Delivered >= 1000 && domainCounter.Bounced == 0 {
		domainCounter.Status = "catch-all"
	} else {
		domainCounter.Status = "not catch-all"
	}

	// Upsert the domain counter
	_, err = domainCountsCollection.UpdateOne(ctx, bson.M{"domain": domain}, bson.M{"$set": domainCounter}, options.Update().SetUpsert(true))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/*
below we use the mux package to extract the domain variable from the URL path, and validate it to
ensure it's not an empty string. We then use the mongo driver package to connect to a MongoDB server running on localhost
at the default port. If there is an error connecting to MongoDB, we return a 500 internal server error.

Once connected to the MongoDB, we use session.DB("catchall").C("domains") to access the domains collection
in the catchall database. We then use the Find method to search for a document with the matching domain field
and use One method to retrieve the matching document.

If there is no matching document found, we return a 404 not found error. If there's an error retrieving the
domain status, we return a 500 internal server error. If all goes well, we prepare a response containing the
status of the domain and write it as JSON to the response writer.
*/

// HandleGetDomainStatus
//Description: This method is responsible for handling the GET requests to the endpoint /domains/{domain}.
//It takes in a request and response writer as input and extracts the domain name from the request.
//It then retrieves the status of the domain from the MongoDB collection and returns it as a JSON response.
//
///*
func HandleGetDomainStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Validate the input
	if domain == "" {
		http.Error(w, "Invalid domain name", http.StatusBadRequest)
		return
	}

	// Create a MongoDB client
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		http.Error(w, "Error connecting to MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(context.TODO())

	// Connect to the MongoDB server
	err = client.Connect(context.TODO())
	if err != nil {
		http.Error(w, "Error connecting to MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the domain status from the database
	var domainCounter DomainCounter
	err = client.Database(DataBase).Collection(Collection).FindOne(context.TODO(), bson.M{"domain": domain}).Decode(&domainCounter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Domain not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error retrieving domain status: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Prepare the response
	response := struct {
		Status string `json:"status"`
	}{
		Status: domainCounter.Status,
	}

	// Write the response as JSON
	json.NewEncoder(w).Encode(response)
}

// HandleBouncedEvent
//handleBouncedEvent: This method is responsible for handling the PUT requests to the endpoint /events/{domain}/bounced.
//It takes in a request and response writer as input and extracts the domain name from the request.
//It then increments the "bounced" count for the domain in the MongoDB collection and updates the status of the domain to "not catch-all".
///*
func HandleBouncedEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// Create a MongoDB client
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		http.Error(w, "Error connecting to MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(context.TODO())

	// Connect to the MongoDB server
	err = client.Connect(context.TODO())
	if err != nil {
		http.Error(w, "Error connecting to MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Use the test database
	db := client.Database(DataBase)

	// Find the domain counter in the collection
	var domainCounter DomainCounter
	err = db.Collection(Collection).FindOne(context.TODO(), bson.M{"domain": domain}).Decode(&domainCounter)
	if err == mgo.ErrNotFound {
		// Initialize the domain counter if it doesn't exist
		domainCounter = DomainCounter{
			Delivered: 0,
			Bounced:   0,
			Status:    "unknown",
		}
	} else if err != nil {
		http.Error(w, "Error finding domain in MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Increment the bounced count
	domainCounter.Bounced++

	// Update the status to not catch-all
	domainCounter.Status = "not catch-all"
	opts := options.Update().SetUpsert(true)
	update := bson.M{
		"$set": &domainCounter,
	}
	// Save the updated domain counter to the collection
	_, err = db.Collection(Collection).UpdateOne(context.TODO(), bson.M{"domain": domain}, update, opts)
	if err != nil {
		http.Error(w, "Error updating domain in MongoDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
