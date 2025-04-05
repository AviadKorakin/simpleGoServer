package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"WebMVCEmployees/config"
	"WebMVCEmployees/controllers"
	"WebMVCEmployees/repository"
	"WebMVCEmployees/router"
	"WebMVCEmployees/services"

	docker "github.com/docker/docker/client" // import the official Docker client package
)

// checkDocker pings the Docker daemon to verify it's running.
func checkDocker() error {
	cli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	_, err = cli.Ping(context.Background())
	return err
}

func main() {
	// Validate that Docker is running.
	if err := checkDocker(); err != nil {
		log.Println("Docker does not appear to be running. Please ensure Docker is installed and started.")
		// Optionally, you can exit or handle the situation differently:
		os.Exit(1)
	}

	// Start the MongoDB container using docker-compose if it's not running.
	if err := config.StartContainers(); err != nil {
		log.Fatal("Failed to start MongoDB container:", err)
	}

	// Connect to MongoDB using our config method.
	client, _, cancel, err := config.ConnectMongo("mongodb://root:example@localhost:27017")
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	// Initialize the EmployeeRepository.
	repo, err := repository.NewEmployeeRepository(client, "employees", "employees")
	if err != nil {
		log.Fatal("Failed to create employee repository:", err)
	}

	// Create the EmployeeService using the repository.
	empService := services.NewEmployeeService(repo)

	// Create the EmployeeController by passing the EmployeeService.
	empController := controllers.NewEmployeeController(empService)

	// Setup the server using our helper function.
	srv := router.SetupServer(empController)

	// Channel to listen for interrupt or termination signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine.
	go func() {
		log.Println("Server is running on port 8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s", err)
		}
	}()

	// Block until a shutdown signal is received.
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for the shutdown process.
	ctxShutdown, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	// Disconnect from MongoDB and stop the container.
	bgCtx, bgCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer bgCancel()

	// Clean up the MongoDB database before disconnecting.
	err = config.CleanMongoDB(client, "employees", bgCtx)
	if err != nil {
		log.Printf("Error cleaning MongoDB: %v", err)
	}

	if err := config.DisconnectMongo(client, bgCtx); err != nil {
		log.Fatal("Error during disconnecting MongoDB:", err)
	}

	log.Println("Server exiting gracefully.")
}
