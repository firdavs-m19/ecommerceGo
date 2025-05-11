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

var productCollection *mongo.Collection

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
	productCollection = client.Database("go_microservices").Collection("products")
}

func grpcDial() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	return conn
}

// HTTP Handlers (Create, Get, List, Update, Delete)
func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var req pb.CreateProductRequest
	json.NewDecoder(r.Body).Decode(&req)
	resp, err := pb.NewProductServiceClient(grpcDial()).CreateProduct(context.Background(), &req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func GetProductHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	resp, err := pb.NewProductServiceClient(grpcDial()).GetProduct(context.Background(), &pb.GetProductRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := pb.NewProductServiceClient(grpcDial()).GetProducts(context.Background(), &pb.GetProductsRequest{})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	var req pb.UpdateProductRequest
	json.NewDecoder(r.Body).Decode(&req)
	resp, err := pb.NewProductServiceClient(grpcDial()).UpdateProduct(context.Background(), &req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	resp, err := pb.NewProductServiceClient(grpcDial()).DeleteProduct(context.Background(), &pb.DeleteProductRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	InitMongo()

	// gRPC server
	lis, err := net.Listen("tcp", ":50052") // Use a different port than user-service
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterProductServiceServer(grpcServer, &ProductServiceServer{}) // Register the gRPC service

	go func() {
		log.Println("gRPC server started on port :50052")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// HTTP routes
	r := mux.NewRouter()
	r.HandleFunc("/api/products", CreateProductHandler).Methods("POST")
	r.HandleFunc("/api/products/{id}", GetProductHandler).Methods("GET")
	r.HandleFunc("/api/products", GetProductsHandler).Methods("GET")
	r.HandleFunc("/api/products/{id}", UpdateProductHandler).Methods("PUT")
	r.HandleFunc("/api/products/{id}", DeleteProductHandler).Methods("DELETE")

	log.Println("HTTP server started at :8081")
	http.ListenAndServe(":8081", r)
}
