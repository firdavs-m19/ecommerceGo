package main

import (
	"context"
	pb "goFinalProject/proto/proto"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductServiceServer struct {
	pb.UnimplementedProductServiceServer
}

func toStringSlice(a interface{}) []string {
	arr, ok := a.(primitive.A)
	if !ok {
		return nil
	}
	var strSlice []string
	for _, v := range arr {
		if s, ok := v.(string); ok {
			strSlice = append(strSlice, s)
		}
	}
	return strSlice
}

func (s *ProductServiceServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	now := time.Now()
	product := bson.M{
		"name":         req.Name,
		"description":  req.Description,
		"price":        req.Price,
		"category":     req.Category,
		"stock":        req.Stock,
		"images":       req.Images,
		"is_available": req.IsAvailable,
		"created_at":   now,
		"updated_at":   now,
	}

	res, err := productCollection.InsertOne(ctx, product)
	if err != nil {
		return nil, err
	}

	oid := res.InsertedID.(primitive.ObjectID)
	return &pb.ProductResponse{
		Id:          oid.Hex(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Stock:       req.Stock,
		Images:      req.Images,
		IsAvailable: req.IsAvailable,
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}, nil
}

func (s *ProductServiceServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	var product bson.M
	err = productCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &pb.ProductResponse{
		Id:          oid.Hex(),
		Name:        product["name"].(string),
		Description: product["description"].(string),
		Price:       product["price"].(float64),
		Category:    product["category"].(string),
		Stock:       int32(product["stock"].(int32)),
		Images:      toStringSlice(product["images"]),
		IsAvailable: product["is_available"].(bool),
		CreatedAt:   product["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:   product["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *ProductServiceServer) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.ProductsResponse, error) {
	cursor, err := productCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*pb.ProductResponse
	for cursor.Next(ctx) {
		var product bson.M
		if err := cursor.Decode(&product); err != nil {
			continue
		}
		products = append(products, &pb.ProductResponse{
			Id:          product["_id"].(primitive.ObjectID).Hex(),
			Name:        product["name"].(string),
			Description: product["description"].(string),
			Price:       product["price"].(float64),
			Category:    product["category"].(string),
			Stock:       int32(product["stock"].(int32)),
			Images:      toStringSlice(product["images"]),
			IsAvailable: product["is_available"].(bool),
			CreatedAt:   product["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
			UpdatedAt:   product["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		})
	}

	return &pb.ProductsResponse{Products: products}, nil
}

func (s *ProductServiceServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"name":         req.Name,
			"description":  req.Description,
			"price":        req.Price,
			"category":     req.Category,
			"stock":        req.Stock,
			"images":       req.Images,
			"is_available": req.IsAvailable,
			"updated_at":   time.Now(),
		},
	}

	_, err = productCollection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return nil, err
	}

	return &pb.ProductResponse{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Stock:       req.Stock,
		Images:      req.Images,
		IsAvailable: req.IsAvailable,
		// created_at left out since we don't change it here
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *ProductServiceServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	_, err = productCollection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return nil, err
	}

	return &pb.DeleteProductResponse{
		Id:      req.Id,
		Success: true,
	}, nil
}
