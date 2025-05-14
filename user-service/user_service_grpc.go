package main

import (
	"context"
	pb "goFinalProject/proto/proto"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	emailCount, err := userCollection.CountDocuments(ctx, bson.M{"email": req.Email})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error checking email uniqueness: %v", err)
	}
	if emailCount > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "Email already in use")
	}

	usernameCount, err := userCollection.CountDocuments(ctx, bson.M{"username": req.Username})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error checking username uniqueness: %v", err)
	}
	if usernameCount > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "Username already in use")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to hash password: %v", err)
	}

	now := time.Now()

	user := bson.M{
		"name":      req.Name,
		"username":  req.Username,
		"email":     req.Email,
		"password":  string(hashedPassword),
		"phone":     req.Phone,
		"role":      req.Role,
		"createdAt": now,
		"updatedAt": now,
	}

	res, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to insert user: %v", err)
	}

	oid := res.InsertedID.(primitive.ObjectID)

	return &pb.UserResponse{
		Id:        oid.Hex(),
		Name:      req.Name,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Role:      req.Role,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	}, nil
}

func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid ObjectID: %v", err)
	}

	var user bson.M
	err = userCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&user)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "User not found: %v", err)
	}

	name, ok1 := user["name"].(string)
	email, ok2 := user["email"].(string)

	if !ok1 || !ok2 {
		return nil, status.Errorf(codes.Internal, "Invalid user data format")
	}

	return &pb.UserResponse{
		Id:    oid.Hex(),
		Name:  name,
		Email: email,
	}, nil
}

func (s *UserServiceServer) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.UsersResponse, error) {
	cursor, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*pb.UserResponse
	for cursor.Next(ctx) {
		var user bson.M
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}

		oid := user["_id"].(primitive.ObjectID)
		users = append(users, &pb.UserResponse{
			Id:    oid.Hex(),
			Name:  user["name"].(string),
			Email: user["email"].(string),
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return &pb.UsersResponse{
		Users: users,
	}, nil
}

func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"name":  req.Name,
			"email": req.Email,
		},
	}

	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return nil, err
	}

	return &pb.UserResponse{
		Id:    oid.Hex(),
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, err
	}

	_, err = userCollection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return nil, err
	}

	return &pb.DeleteUserResponse{
		Id:      oid.Hex(),
		Success: true,
	}, nil
}
