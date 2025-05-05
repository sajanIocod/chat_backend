package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"password" bson:"password"`
}

type Response struct {
	ResponseCode int         `json:"responseCode"`
	Message      string      `json:"message"`
	Data         interface{} `json:"data"`
}

type UserResponse struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email    string             `json:"email" bson:"email"`
	Token    string             `json:"token"`
	Username string             `json:"username" bson:"username"`
}
