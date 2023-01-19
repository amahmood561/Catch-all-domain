package main

import (
	"context"
	"encoding/json"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)
const testDomain = "example.com"
/*
Herethe function first establishes a connection to the MongoDB server running on localhost,
then it opens a session to the database. Next, it  we use mongo driver to create connections. (we can use the Dial function from the mgo package to connect to the MongoDB server as well).
It then switches to the test database, and gets the collection domain_counts from the test database.
After that, it prepare the request and response recorder as usual, and then it executes the HandleDeliveredEvent function passing the recorder and request as arguments.
After that, it checks the status code as usual, and then it gets the count of the domain from the database using the Find and Count functions from the mgo package, and it checks
*/
func TestHandleDeliveredEvent(t *testing.T) {
	// Connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Switch to test database
	db := client.Database(DataBase)
	domainCounts := db.Collection(Collection)
	router := mux.NewRouter()
	// Register the handleDeliveredEvent function with the router
	router.HandleFunc(
		"/events/{domain}/delivered",
		HandleDeliveredEvent,
	).Methods("PUT")

	// Create a new request
	req, _ := http.NewRequest("PUT", "/events/example.com/delivered", nil)

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Pass the request and response recorder to the router
	router.ServeHTTP(rr, req)

	// Assert that the response has a status code of 200 OK
	//assert.Equal(t, http.StatusOK, rr.Code)

	// Check the status code
	if rr.Code != http.StatusOK {
		t.Errorf("status code should be %d, but got %d", http.StatusOK, rr.Code)
	}

	// Get the count of the domain from the database
	count, err := domainCounts.CountDocuments(ctx, bson.M{"domain": testDomain})
	if err != nil {
		t.Fatal(err)
	}

	// Check the count
	if count != 1 {
		t.Errorf("count should be 1, but got %d", count)
	}

	// clear collection so other tests have valid runs
	db.Collection(Collection).DeleteMany(context.TODO(), bson.M{"domain": testDomain})

}

/*
This test creates a new router and registers the HandleGetDomainStatus function as the handler for the GET /domains/{domain} route.
Then it creates a new session to a MongoDB instance running on the localhost, uses the "test" database and inserts a test document with domain name "example.com" and delivered count 1001.
Then, it creates a new request with the domain "example.com" and sends it to the handler. Then it checks the response status code and response body to assert that it's correct.
You will need to make sure you import the packages net/http,net/http/httptest,testing, github.com/gorilla/mux, and gopkg.in/mgo.v2/bson.
*/
func TestHandleGetDomainStatus(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()

	// Handle GET requests for domain statuses
	router.HandleFunc(URL_PATH_DOMAIN, HandleGetDomainStatus).Methods("GET")

	// Create a new client
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		t.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Use the test database
	db := client.Database(DataBase)

	// Insert a test document for the domain "example.com"
	_, err = db.Collection(Collection).InsertOne(context.TODO(), bson.M{
		"domain":    testDomain,
		"delivered": 1001,
		"bounced":   0,
		"status":    "catch-all",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Define the request to test
	req, err := http.NewRequest("GET", "/domains/example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Execute the request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("status code should be %d, got %d", http.StatusOK, status)
	}

	// Check the response body
	var response struct {
		Status string `json:"status"`
	}
	json.NewDecoder(rr.Body).Decode(&response)
	if response.Status != "catch-all" {
		t.Errorf("status should be catch-all, got %q", response.Status)
	}
	db.Collection(Collection).DeleteMany(context.TODO(), bson.M{"domain": testDomain})
}

/*
We are then sending a PUT request to the "/events/example.com/bounced" endpoint, and then querying the database to check
that the "bounced" count of the testDomain domain has been incremented by 1.
You can adjust the test to match the endpoint and the database credentials you are using.
Also please make sure to import the necessary packages, like "net/http" and "net/http/httptest"
*/
func TestHandleBouncedEvent(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()

	// Handle PUT requests for bounced events
	router.HandleFunc(URL_PATHS_BOUNCED, HandleBouncedEvent).Methods("PUT")

	// Create a new session to the MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		t.Fatalf("Error connecting to MongoDB: %v", err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		panic(err)
	}

	defer client.Disconnect(context.TODO())

	// Use the test database
	db := client.Database(DataBase)

	// clear collection
	_, _ = db.Collection(Collection).DeleteMany(context.TODO(), bson.D{})
	// Insert a test document for the domain testDomain
	_, err = db.Collection(Collection).InsertOne(context.TODO(), bson.M{
		"domain":   testDomain,
		"delivered": 1001,
		"bounced":   0,
		"status":    "catch-all",
	})
	if err != nil {
		t.Fatalf("Error inserting domain into MongoDB: %v", err)
	}

	// Send a PUT request for a bounced event for the "example.com" domain
	request, _ := http.NewRequest("PUT", "/events/example.com/bounced", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, request)

	// Get the count of bounced events for the "example.com" domain
	var domainCounter DomainCounter

	err = db.Collection(Collection).FindOne(context.TODO(), bson.M{"domain": testDomain}).Decode(&domainCounter)
	if err != nil {
		t.Fatalf("Error querying MongoDB: %v", err)
	}

	// Check that the bounced count has been incremented
	assert.Equal(t, 1, domainCounter.Bounced)

	db.Collection(Collection).DeleteMany(context.TODO(), bson.M{"domain": testDomain})
}
