package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID   primitive.ObjectID `json:"senderID" bson:"senderID"`
	ReceiverID primitive.ObjectID `json:"receiverID" bson:"receiverID"`
	Content    string             `json:"content" bson:"content"`
	Seen       bool               `json:"seen" bson:"seen"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

type MessageRequest struct {
	ReceiverID string `json:"receiverId" binding:"required"`
	Content    string `json:"content" binding:"required"`
}
