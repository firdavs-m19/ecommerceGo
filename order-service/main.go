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

var orderCollection *mongo.Collection
var userClient pb.UserServiceClient
var productClient pb.ProductServiceClient

type ProductItemInput struct {
	ProductId string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type CreateOrderInput struct {
	UserId   string             `json:"userId"`
	Products []ProductItemInput `json:"products"`
}

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
	orderCollection = client.Database("go_microservices").Collection("orders")
}

func initGRPCClients() {
	userConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	userClient = pb.NewUserServiceClient(userConn)

	productConn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to product-service: %v", err)
	}
	productClient = pb.NewProductServiceClient(productConn)
}

func grpcDial() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	return conn
}

func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var input CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Println("req.UserId is:", input.UserId)

	_, err := userClient.GetUser(context.Background(), &pb.GetUserRequest{Id: input.UserId})
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var productItems []*pb.ProductItem
	var totalPrice float64

	for _, item := range input.Products {
		productResp, err := productClient.GetProduct(context.Background(), &pb.GetProductRequest{Id: item.ProductId})
		if err != nil {
			http.Error(w, "Product not found: "+item.ProductId, http.StatusNotFound)
			return
		}
		productItems = append(productItems, &pb.ProductItem{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		})
		totalPrice += productResp.Price * float64(item.Quantity)
	}

	req := &pb.CreateOrderRequest{
		UserId:     input.UserId,
		Products:   productItems,
		TotalPrice: totalPrice,
	}

	grpcClient := pb.NewOrderServiceClient(grpcDial())
	resp, err := grpcClient.CreateOrder(context.Background(), req)
	if err != nil {
		http.Error(w, "Error creating order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	grpcClient := pb.NewOrderServiceClient(grpcDial())
	resp, err := grpcClient.GetOrder(context.Background(), &pb.GetOrderRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
func GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	grpcClient := pb.NewOrderServiceClient(grpcDial())

	resp, err := grpcClient.GetOrders(context.Background(), &pb.GetOrdersRequest{})
	if err != nil {
		http.Error(w, "Failed to fetch orders: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	InitMongo()
	initGRPCClients()

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, &OrderServiceServer{})

	go func() {
		log.Println("gRPC server started at :50053")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/api/orders", CreateOrderHandler).Methods("POST")
	r.HandleFunc("/api/orders/{id}", GetOrderHandler).Methods("GET")
	r.HandleFunc("/api/orders", GetOrdersHandler).Methods("GET")
	http.Handle("/", r)

	log.Println("HTTP server started at :8082")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
