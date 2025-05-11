package main

import (
	"context"

	pb "goFinalProject/proto/proto"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderServiceServer struct {
	pb.UnimplementedOrderServiceServer
}

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	// Convert pb.ProductItem to BSON format
	var productDocs []bson.M
	for _, item := range req.Products {
		productDocs = append(productDocs, bson.M{
			"product_id": item.ProductId,
			"quantity":   item.Quantity,
		})
	}

	order := bson.M{
		"user_id":     req.UserId,
		"products":    productDocs,
		"total_price": req.TotalPrice,
	}

	res, err := orderCollection.InsertOne(ctx, order)
	if err != nil {
		return nil, err
	}

	oid := res.InsertedID.(primitive.ObjectID)

	// Convert back to gRPC response
	var grpcProducts []*pb.ProductItem
	for _, item := range req.Products {
		grpcProducts = append(grpcProducts, &pb.ProductItem{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		})
	}

	return &pb.OrderResponse{
		Id:         oid.Hex(),
		UserId:     req.UserId,
		Products:   grpcProducts,
		TotalPrice: req.TotalPrice,
	}, nil
}

func (s *OrderServiceServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	var order bson.M
	err = orderCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&order)
	if err != nil {
		return nil, err
	}

	rawProducts := order["products"].(primitive.A)
	var grpcProducts []*pb.ProductItem
	for _, p := range rawProducts {
		productMap := p.(primitive.M)
		grpcProducts = append(grpcProducts, &pb.ProductItem{
			ProductId: productMap["product_id"].(string),
			Quantity:  int32(productMap["quantity"].(int32)),
		})
	}

	return &pb.OrderResponse{
		Id:         oid.Hex(),
		UserId:     order["user_id"].(string),
		Products:   grpcProducts,
		TotalPrice: order["total_price"].(float64),
	}, nil
}

func (s *OrderServiceServer) GetOrders(ctx context.Context, req *pb.GetOrdersRequest) (*pb.OrdersResponse, error) {
	cursor, err := orderCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*pb.OrderResponse

	for cursor.Next(ctx) {
		var order bson.M
		if err := cursor.Decode(&order); err != nil {
			return nil, err
		}

		oid := order["_id"].(primitive.ObjectID)
		userId := order["user_id"].(string)
		totalPrice := order["total_price"].(float64)

		// Convert products from BSON to []*pb.ProductItem
		var grpcProducts []*pb.ProductItem
		if rawProducts, ok := order["products"].(primitive.A); ok {
			for _, p := range rawProducts {
				productMap := p.(primitive.M)
				grpcProducts = append(grpcProducts, &pb.ProductItem{
					ProductId: productMap["product_id"].(string),
					Quantity:  int32(productMap["quantity"].(int32)),
				})
			}
		}

		orders = append(orders, &pb.OrderResponse{
			Id:         oid.Hex(),
			UserId:     userId,
			Products:   grpcProducts,
			TotalPrice: totalPrice,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return &pb.OrdersResponse{Orders: orders}, nil
}
