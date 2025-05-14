package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "goFinalProject/proto/proto"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

var userCollection *mongo.Collection

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

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

	userCollection = client.Database("go_microservices").Collection("users")
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var userReq pb.CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.CreateUser(context.Background(), &userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["id"]

	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.GetUser(context.Background(), &pb.GetUserRequest{Id: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.GetUsers(context.Background(), &pb.GetUsersRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	var userReq pb.UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get ID from URL and set it in the request
	params := mux.Vars(r)
	userID := params["id"]
	userReq.Id = userID

	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.UpdateUser(context.Background(), &userReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["id"]

	grpcClient := pb.NewUserServiceClient(grpcDial())
	resp, err := grpcClient.DeleteUser(context.Background(), &pb.DeleteUserRequest{Id: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func grpcDial() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	return conn
}

func main() {
	InitMongo()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &UserServiceServer{})
	go func() {
		log.Println("gRPC server started on port :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/api/users", CreateUserHandler).Methods("POST")
	r.HandleFunc("/api/users/{id}", GetUserHandler).Methods("GET")
	r.HandleFunc("/api/users", GetAllUsersHandler).Methods("GET")
	r.HandleFunc("/api/users/{id}", UpdateUserHandler).Methods("PUT")
	r.HandleFunc("/api/users/{id}", DeleteUserHandler).Methods("DELETE")

	http.Handle("/", r)

	log.Println("HTTP server started on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
