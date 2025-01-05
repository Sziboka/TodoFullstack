package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection mongo.Collection

type CustomHandler struct{}

func (h *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l := *log.Default()

	switch r.URL.Path {
	case "/Create":
		if r.Method == http.MethodPost {
			fmt.Printf("Creating Task with the given req\n")
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}

	case "/Read":
		if r.Method == http.MethodGet {
			t, _ := ReadID(1, l)
			fmt.Fprintf(w, out(t))
		}

	case "/ReadAll":
		if r.Method == http.MethodGet {
			ReadAll(l)
		}

	case "/Update":
		if r.Method == http.MethodPost {
			fmt.Printf("Updating the Task with the given id with the given data\n")
		}

	case "/Delete":
		if r.Method == http.MethodGet {
			fmt.Printf("Deleting the Task with the given ID\n")
		}

	case "/DeleteAll":
		if r.Method == http.MethodGet {
			fmt.Printf("Deleting all the Tasks\n")
		}

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func main() {
	client, err := connectDB(
		"mongodb://localhost:27017",
		"admin",
		"password",
		"admin",
	)

	if err != nil {
		fmt.Print(err)
	}

	db := client.Database("TaskDB")
	collection = *db.Collection("TaskCollection")
	fmt.Println("Using database:", db.Name(), "and collection:", collection.Name())
	//test := Task{
	//	ID:      1,
	//	Name:    "testTask",
	//	Context: "Creating a test task",
	//	Ready:   true,
	//}
	//Create(test, *l)
	//ReadID(1, *l)
	l := *log.Default()

	StartAPi(":9090", l)

}

// FORM
func out(t Task) string {
	return fmt.Sprintf("ID: %d\nName: %s\nContext: %s\nReady: %t\n", t.ID, t.Name, t.Context, t.Ready)
}

// STRUCT
type Task struct {
	ID      int    `bson:"id"`
	Name    string `bson:"name"`
	Context string `bson:"context"`
	Ready   bool   `bson:"ready"`
}

// CRUDS
func Create(t Task, l log.Logger) error {
	_, err := collection.InsertOne(context.TODO(), t)
	if err != nil {
		return err
	}
	l.Printf("Task created with the following data:\n%s\n", out(t))
	return nil
}

func ReadID(ID int, l log.Logger) (Task, error) {
	var result Task
	filter := bson.M{"id": ID}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return Task{}, err
	}
	l.Printf("Found Task with the given ID of %d\n%s\n", ID, out(result))
	return result, nil
}

func ReadAll(l log.Logger) ([]Task, error) {
	filter := bson.D{}
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("error finding documents: %v", err)
	}

	var tasks []Task

	for cursor.Next(context.TODO()) {
		var task Task
		err := cursor.Decode(&task)
		if err != nil {
			return nil, fmt.Errorf("error decoding document: %v", err)
		}
		tasks = append(tasks, task)
	}
	l.Printf("Succesful Quarry")
	return tasks, nil
}

func Update(ID int, new Task, l log.Logger) (string, error) {
	return "", nil
}

func Delete(ID int, l log.Logger) (string, error) {
	return "", nil
}

func DeleteAll(l log.Logger) (string, error) {
	return "", nil
}

func StartAPi(port string, l log.Logger) error {
	// Create a custom handler
	handler := &CustomHandler{}

	// Configure the HTTP server
	server := &http.Server{
		Addr:    port,    // Address and port to listen on
		Handler: handler, // Custom handler
	}
	l.Printf("Starting server on localhost%s\n", port)
	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func connectDB(URI string, username string, password string, authSource string) (*mongo.Client, error) {
	credentials := options.Credential{
		Username:   username,
		Password:   password,
		AuthSource: authSource,
	}

	// MongoDB connection URI
	mongoURI := URI // Replace with your URI

	// Set client options
	clientOptions := options.Client().ApplyURI(mongoURI).SetAuth(credentials)

	// Connect to MongoDB
	client, _ := mongo.Connect(context.TODO(), clientOptions)
	//if err != nil {
	//	return err
	//}

	// Check the connection
	_ = client.Ping(context.TODO(), nil)
	//if err != nil {
	//	return err
	//}

	fmt.Println("Connected to MongoDB!")
	return client, nil
}
