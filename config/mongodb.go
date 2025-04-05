package config

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// isContainerRunning checks if a Docker container with the given name is running.
func IsContainerRunning(containerName string) (bool, error) {
	// Run "docker ps" filtering by container name.
	cmd := exec.Command("docker", "ps", "--filter", "name="+containerName, "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	// If the output contains the containerName, it's running.
	return strings.Contains(string(output), containerName), nil
}

// startMongoContainer starts the MongoDB container using docker compose if it's not already running.
func StartMongoContainer() error {
	const containerName = "webmvc_employees_mongodb"
	running, err := IsContainerRunning(containerName)
	if err != nil {
		return err
	}
	if running {
		log.Println("MongoDB container is already running.")
		return nil
	}

	log.Println("Starting MongoDB container via docker compose...")
	// Change the working directory if necessary so that docker-compose.yml is found.
	cmd := exec.Command("docker", "compose", "up", "-d", "mongodb")
	cmd.Dir = "." // Adjust this if your docker-compose.yml is located elsewhere.
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error starting docker compose:", string(output))
		return err
	}
	return nil
}

// stopMongoContainer stops the MongoDB container using docker compose.
func StopMongoContainer() error {
	log.Println("Stopping MongoDB container via docker compose...")
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = "." // Adjust this if necessary.
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error stopping docker compose:", string(output))
		return err
	}
	return nil
}

func StartContainers() error {
	log.Println("Starting containers via docker compose...")
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error starting docker compose:", string(output))
		return err
	}
	return nil
}

// StopContainers stops all containers defined in the docker-compose.yml file using docker compose.
func StopContainers() error {
	log.Println("Stopping containers via docker compose...")
	cmd := exec.Command("docker", "compose", "stop")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error stopping docker compose:", string(output))
		return err
	}
	return nil
}

// CleanupContainers stops and removes all containers defined in your docker-compose file using docker compose.
func CleanupContainers() error {
	log.Println("Cleaning up containers via docker compose (down)...")
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = "." // Adjust if necessary.
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error cleaning up containers:", string(output))
		return err
	}
	return nil
}

func ConnectMongo(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {
	// Create a context with a 10-second timeout for operations.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Apply the URI to the client options.
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to MongoDB using the client options.
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		// Cancel the context to release resources before returning the error.
		cancel()
		return nil, nil, nil, err
	}

	// Return the client, context, and cancel function.
	return client, ctx, cancel, nil
}

// DisconnectMongo disconnects the MongoDB client after cleaning the database and stopping containers.
func DisconnectMongo(client *mongo.Client, ctx context.Context) error {
	// Disconnect from MongoDB.
	if err := client.Disconnect(ctx); err != nil {
		return err
	}
	if err := StopContainers(); err != nil {
		return err
	}
	return nil
}

func CleanMongoDB(client *mongo.Client, dbName string, ctx context.Context) error {
	log.Println("Cleaning up MongoDB database:", dbName)
	dropCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // Increased timeout
	defer cancel()
	err := client.Database(dbName).Drop(dropCtx)
	if err != nil {
		log.Printf("Error dropping database %s: %v", dbName, err)
	}
	return err
}
