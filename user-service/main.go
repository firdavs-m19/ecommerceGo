package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "goFinalProject/proto/proto" // Adjust path to your pb file

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

var userCollection *mongo.Collection // Declare it globally only once

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

// Initialize MongoDB client
func InitMongo() {
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	userCollection = client.Database("go_microservices").Collection("users") // Assign to the global variable
}

// HTTP handler for CreateUser
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var userReq pb.CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call the gRPC CreateUser method
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.CreateUser(context.Background(), &userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HTTP handler for GetUser
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["id"]

	// Call the gRPC GetUser method
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.GetUser(context.Background(), &pb.GetUserRequest{Id: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HTTP handler for GetAllUsers
func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Call the gRPC GetUsers method
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.GetUsers(context.Background(), &pb.GetUsersRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HTTP handler for UpdateUser
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	var userReq pb.UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call the gRPC UpdateUser method
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.UpdateUser(context.Background(), &userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HTTP handler for DeleteUser
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["id"]

	// Call the gRPC DeleteUser method
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.DeleteUser(context.Background(), &pb.DeleteUserRequest{Id: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// gRPC dial function
func grpcDial() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	return conn
}

func main() {
	// Initialize MongoDB
	InitMongo()

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &UserServiceServer{}) // Register gRPC server
	go func() {
		log.Println("gRPC server started on port :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Set up HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/api/users", CreateUserHandler).Methods("POST")
	r.HandleFunc("/api/users/{id}", GetUserHandler).Methods("GET")
	r.HandleFunc("/api/users", GetAllUsersHandler).Methods("GET")        // New endpoint for getting all users
	r.HandleFunc("/api/users/{id}", UpdateUserHandler).Methods("PUT")    // New endpoint for updating a user
	r.HandleFunc("/api/users/{id}", DeleteUserHandler).Methods("DELETE") // New endpoint for deleting a user

	http.Handle("/", r)

	// Start HTTP server
	log.Println("HTTP server started on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
