package utils

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ObjectIDFromHex(hex string) primitive.ObjectID {
	id, _ := primitive.ObjectIDFromHex(hex)
	return id
}
