package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection mongo.Collection

type CustomHandler struct{}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (h *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	l := *log.Default()

	type postReq struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	type readReq struct {
		ID int `json:"id"`
	}

	type readAllResult struct {
		Tasks []Task `json:"tasks"`
	}

	switch r.URL.Path {
	case "/Create":
		if r.Method == http.MethodPost {
			l.Printf("%s : %s\n", r.Method, r.URL.Path)
			var task Task
			err := json.NewDecoder(r.Body).Decode(&task)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			}
			status, _ := Create(task, &l)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(postReq{Status: status})
		}

	case "/Read":
		if r.Method == http.MethodGet {
			l.Printf("%s : %s\n", r.Method, r.URL.Path)
			// Read the ID from the query parameters instead of the body
			idParam := r.URL.Query().Get("id")
			if idParam == "" {
				json.NewEncoder(w).Encode(postReq{Error: "ID is required"})
				return
			}

			// Convert the ID to an integer
			ID, err := strconv.Atoi(idParam)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: "Invalid ID format"})
				return
			}

			task, _, _ := ReadID(ID, &l)
			if isTaskEmpty(task) {
				json.NewEncoder(w).Encode(postReq{Error: "Task not found"})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(task)
		}

	case "/ReadAll":
		if r.Method == http.MethodGet {
			l.Printf("%s : %s\n", r.Method, r.URL.Path)
			var tasks []Task
			tasks, err := ReadAll(&l)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(readAllResult{Tasks: tasks})
		}

	case "/Update":
		if r.Method == http.MethodPost {
			l.Printf("%s : %s\n", r.Method, r.URL.Path)
			var newTask Task
			err := json.NewDecoder(r.Body).Decode(&newTask)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			}
			status, err := Update(newTask, &l)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(postReq{Status: status})
		}

	case "/Delete":
		if r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		l.Printf("%s : %s\n", r.Method, r.URL.Path)

		// Read the ID from the query parameters instead of the body
		idParam := r.URL.Query().Get("id")
		if idParam == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(postReq{Error: "ID is required"})
			return
		}

		// Convert the ID to an integer
		ID, err := strconv.Atoi(idParam)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(postReq{Error: "Invalid ID format"})
			return
		}

		// Call the Delete function with the parsed ID
		status, err := Delete(ID, &l)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(postReq{Status: status})

	case "/DeleteAll":
		if r.Method == http.MethodDelete {
			l.Printf("%s : %s\n", r.Method, r.URL.Path)
			status, err := DeleteAll(&l)
			if err != nil {
				json.NewEncoder(w).Encode(postReq{Error: err.Error()})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(postReq{Status: status})
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
	l := *log.Default()
	StartAPi(":9090", &l)
}

// FORM
func out(t Task) string {
	return fmt.Sprintf("ID: %d\nName: %s\nContext: %s\nReady: %t\n", t.ID, t.Name, t.Context, t.Ready)
}

// STRUCT
type Task struct {
	ID      int    `bson:"id" json:"id"`
	Name    string `bson:"name" json:"name"`
	Context string `bson:"context" json:"context"`
	Ready   bool   `bson:"ready" json:"ready"`
}

func isTaskEmpty(task Task) bool {
	return task.ID == 0 && task.Name == "" && task.Context == "" && !task.Ready
}

// CRUDS
func Create(t Task, l *log.Logger) (string, error) {
	_, _, err := ReadID(t.ID, l)
	if err == nil {
		return fmt.Sprintf("Item with given id of %d already exists", t.ID), err
	} else {
		_, err = collection.InsertOne(context.TODO(), t)
		if err != nil {
			return "Error while insert", err
		}
		if isTaskEmpty(t) {
			l.Println("Error, empty Task")
			return "", fmt.Errorf("error,empty task")
		}
		l.Printf("Task created with the following data:\n%s\n", out(t))
		return fmt.Sprintf("Task created with the following id: %d", t.ID), nil
	}

}

func ReadID(ID int, l *log.Logger) (Task, string, error) {
	var result Task
	filter := bson.M{"id": ID}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		l.Printf("Cant read item with ID of %d\n", ID)
		return Task{}, fmt.Sprintf("Cant read item with ID of %d", ID), err
	}
	l.Printf("Found Task with the given ID of %d\n%s\n", ID, out(result))
	return result, fmt.Sprintf("Item Read with id of %d", ID), nil
}

func ReadAll(l *log.Logger) ([]Task, error) {
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

//type Task struct {
//	ID      int    `bson:"id"`
//	Name    string `bson:"name"`
//	Context string `bson:"context"`
//	Ready   bool   `bson:"ready"`
//}

func Update(newTask Task, l *log.Logger) (string, error) {
	// Define the filter and update
	filter := bson.M{"id": newTask.ID}
	update := bson.M{"$set": bson.M{
		"name":    newTask.Name,
		"context": newTask.Context,
		"ready":   newTask.Ready,
	}}

	// Perform the update
	result, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		l.Printf("Error updating task: %v\n", err)
		return "", err
	}

	if result.MatchedCount == 0 {
		return "", fmt.Errorf("no task found with ID %d", newTask.ID)
	}
	l.Printf("Task with the ID of %d is updated:\n%s", newTask.ID, out(newTask))

	return fmt.Sprintf("Task with ID %d updated successfully", newTask.ID), nil
}

func Delete(ID int, l *log.Logger) (string, error) {
	filter := bson.M{"id": ID}

	// Perform the DeleteOne operation
	deleteResult, err := collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		l.Printf("Error deleting document with ID %d: %v", ID, err)
		return "", err
	}

	// Check if any document was deleted
	if deleteResult.DeletedCount == 0 {
		l.Printf("No document found with ID %d to delete", ID)
		return "", fmt.Errorf("no document found with ID %d", ID)
	}

	// Return success message
	l.Printf("Successfully deleted document with ID %d", ID)
	return "Document successfully deleted", nil
}

func DeleteAll(l *log.Logger) (string, error) {
	_, err := collection.DeleteMany(context.TODO(), bson.M{})
	if err != nil {
		l.Println(err)
		return "", err
	}
	l.Println("Collection deleted")
	return "Collection deleted", nil
}

func StartAPi(port string, l *log.Logger) error {
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
