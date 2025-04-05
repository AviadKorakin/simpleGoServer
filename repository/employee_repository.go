package repository

import (
	"WebMVCEmployees/models"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EmployeeRepository encapsulates operations on the employee collection.
type EmployeeRepository struct {
	Collection *mongo.Collection
}

// NewEmployeeRepository creates a new EmployeeRepository and ensures that a unique index is set on the email field.
func NewEmployeeRepository(client *mongo.Client, dbName, collName string) (*EmployeeRepository, error) {
	coll := client.Database(dbName).Collection(collName)

	// Create a unique index on the email field.
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: models.EmployeeRef.Email, Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Failed to create unique index on email: %v", err)
		return nil, err
	}

	return &EmployeeRepository{
		Collection: coll,
	}, nil
}
