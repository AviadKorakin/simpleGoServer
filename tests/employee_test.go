package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"WebMVCEmployees/config"
	"WebMVCEmployees/controllers"
	"WebMVCEmployees/models"
	"WebMVCEmployees/repository"
	"WebMVCEmployees/router"
	"WebMVCEmployees/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	docker "github.com/docker/docker/client"
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

var testServer *httptest.Server

// TestMain is executed before any tests run.
func TestMain(m *testing.M) {
	// Load environment variables from .env.test.
	if err := godotenv.Load(".env.test"); err != nil {
		log.Println("No .env.test file found, continuing with system environment variables")
	}

	// Validate that Docker is running.
	if err := checkDocker(); err != nil {
		log.Println("Docker does not appear to be running. Please ensure Docker is installed and started.")
		os.Exit(1)
	}

	// Set Gin to test mode.
	gin.SetMode(gin.TestMode)

	// Start the MongoDB container using docker-compose if it's not running.
	if err := config.StartContainers(); err != nil {
		log.Fatal("Failed to start MongoDB container:", err)
	}

	// Retrieve MongoDB connection settings from environment variables.
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		log.Fatal("MONGO_URL environment variable not set")
	}
	mongoDB := os.Getenv("MONGO_DB")
	if mongoDB == "" {
		mongoDB = "employees"
	}
	mongoCollection := os.Getenv("MONGO_COLLECTION")
	if mongoCollection == "" {
		mongoCollection = "employees"
	}

	// Connect to MongoDB using our config method.
	client, ctx, cancel, err := config.ConnectMongo(mongoURL)
	if err != nil {
		panic("failed to connect to mongo: " + err.Error())
	}
	defer cancel()

	// Initialize the EmployeeRepository.
	repo, err := repository.NewEmployeeRepository(client, mongoDB, mongoCollection)
	if err != nil {
		log.Fatal("Failed to create employee repository:", err)
	}

	// Create the EmployeeService using the repository.
	empService := services.NewEmployeeService(repo)
	empController := controllers.NewEmployeeController(empService)

	// Setup the router.
	r := router.SetupRouter(empController)

	// Launch the test server once for all tests.
	testServer = httptest.NewServer(r)

	// Run all tests.
	code := m.Run()

	// Clean up the test server.
	testServer.Close()

	// Clean up the MongoDB database before disconnecting.
	err = config.CleanMongoDB(client, mongoDB, ctx)
	if err != nil {
		log.Printf("Error cleaning MongoDB: %v", err)
	}

	// Disconnect from MongoDB.
	config.DisconnectMongo(client, ctx)

	// Exit with the proper code.
	os.Exit(code)
}

func TestE2E_CreateEmployee(t *testing.T) {
	// Create a new employee payload.
	newEmployee := models.Employee{
		Email: "test2@example.com",
		Name:  "Test User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var empResp models.EmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&empResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if empResp.Email != newEmployee.Email {
		t.Errorf("expected email %s, got %s", newEmployee.Email, empResp.Email)
	} else {
		t.Log("TestE2E_CreateEmployee passed")
	}
}

func TestE2E_CreateEmployee_InvalidPassword(t *testing.T) {
	// Invalid password: "aaa" does not meet the requirement.
	newEmployee := models.Employee{
		Email: "invalidpassword@example.com",
		Name:  "Invalid Password User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "aaa",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid password, got %d", resp.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_InvalidPassword passed")
	}
}

func TestE2E_CreateEmployee_InvalidBirthdate(t *testing.T) {
	// Invalid birthdate: Day provided as "3" instead of "03".
	newEmployee := models.Employee{
		Email: "invalidbirthday@example.com",
		Name:  "Invalid Birthday User",
		Birthdate: models.Birthdate{
			Day:   "3", // Invalid: should be "03"
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid birthdate, got %d", resp.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_InvalidBirthdate passed")
	}
}
func TestE2E_CreateEmployee_PasswordTooShort(t *testing.T) {
	// Invalid password: "T1" is only 2 characters.
	newEmployee := models.Employee{
		Email: "passwordtooshort@example.com",
		Name:  "Password Too Short",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "T1",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for password too short, got %d", resp.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_PasswordTooShort passed")
	}
}

func TestE2E_CreateEmployee_PasswordNoDigit(t *testing.T) {
	// Invalid password: "Test" has no digit.
	newEmployee := models.Employee{
		Email: "passwordnodigit@example.com",
		Name:  "Password No Digit",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for password with no digit, got %d", resp.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_PasswordNoDigit passed")
	}
}

func TestE2E_CreateEmployee_PasswordNoUpperCase(t *testing.T) {
	// Invalid password: "test1" has a digit but no uppercase letter.
	newEmployee := models.Employee{
		Email: "passwordnouppercase@example.com",
		Name:  "Password No UpperCase",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "test1",
	}
	body, _ := json.Marshal(newEmployee)
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for password with no uppercase, got %d", resp.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_PasswordNoUpperCase passed")
	}
}

func TestE2E_GetEmployee_Success(t *testing.T) {
	// First, create an employee using the POST endpoint.
	newEmployee := models.Employee{
		Email: "loginSuccess@example.com",
		Name:  "Login Success User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, err := json.Marshal(newEmployee)
	if err != nil {
		t.Fatalf("failed to marshal employee: %v", err)
	}
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	defer resp.Body.Close()

	// Log POST response
	postBody, _ := io.ReadAll(resp.Body)
	t.Logf("POST response body: %s", string(postBody))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to create employee, expected status 200, got %d", resp.StatusCode)
	}

	// Now, send a GET request with the correct email and password.
	getURL := testServer.URL + "/employees/" + newEmployee.Email + "?password=" + newEmployee.Password
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request: %v", err)
	}
	defer getResp.Body.Close()

	// Read and log GET response
	getBody, _ := io.ReadAll(getResp.Body)
	t.Logf("GET response body: %s", string(getBody))
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for GET, got %d", getResp.StatusCode)
	}

	// Decode the response from a bytes.Reader since we've already read the body.
	var empResp models.EmployeeResponse
	if err := json.NewDecoder(bytes.NewReader(getBody)).Decode(&empResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	if empResp.Email != newEmployee.Email {
		t.Errorf("expected email %s, got %s", newEmployee.Email, empResp.Email)
	} else {
		t.Log("TestE2E_GetEmployee_Success passed")
	}
}

func TestE2E_GetEmployee_NotFound(t *testing.T) {
	// Attempt to get an employee that doesn't exist.
	getURL := testServer.URL + "/employees/nonexistent@example.com?password=Test1"
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 for non-existent employee, got %d", getResp.StatusCode)
	} else {
		t.Log("TestE2E_GetEmployee_NotFound passed")
	}
}
func TestGetEmployeeHandler_PasswordNotExposed(t *testing.T) {
	// First, create an employee with a known password.
	newEmployee := models.Employee{
		Email: "testpass@example.com",
		Name:  "Test Password User",
		Birthdate: models.Birthdate{
			Day:   "15",
			Month: "05",
			Year:  "1995",
		},
		Roles:    []string{"Tester"},
		Manager:  nil,
		Password: "Secret123",
	}
	body, err := json.Marshal(newEmployee)
	if err != nil {
		t.Fatalf("Failed to marshal employee: %v", err)
	}
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create employee: %v", err)
	}
	defer resp.Body.Close()

	// Verify that the creation was successful.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 on POST, got %d", resp.StatusCode)
	}

	// Now, GET the employee using the correct email and password.
	getURL := testServer.URL + "/employees/" + newEmployee.Email + "?password=" + newEmployee.Password
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 on GET, got %d", getResp.StatusCode)
	}

	// Decode the JSON response into a map.
	var result map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode GET response: %v", err)
	}

	// Check that the "password" key is either not present or its value is an empty string.
	if pwd, exists := result["password"]; exists {
		if str, ok := pwd.(string); ok && str != "" {
			t.Errorf("Expected password field to be omitted or empty, got %q", str)
		}
	} else {
		t.Log("Password field is not present in the response, as expected.")
	}

	t.Log("TestGetEmployeeHandler_PasswordNotExposed passed")
}
func TestE2E_ListEmployees_Pagination(t *testing.T) {
	// First, create 10 employees.
	totalEmployees := 10
	for i := 1; i <= totalEmployees; i++ {
		emp := models.Employee{
			Email: fmt.Sprintf("employee%d@example.com", i),
			Name:  fmt.Sprintf("Employee %d", i),
			Birthdate: models.Birthdate{
				Day:   "01",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Developer"},
			Manager:  nil,
			Password: "Test1",
		}
		body, err := json.Marshal(emp)
		if err != nil {
			t.Fatalf("failed to marshal employee %d: %v", i, err)
		}
		resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("failed to create employee %d: %v", i, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for employee %d, got %d", i, resp.StatusCode)
		}
	}

	// Now test pagination: request page=1, size=5.
	getURL := testServer.URL + "/employees?page=1&size=5"
	resp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("GET request failed for page 1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for page 1, got %d", resp.StatusCode)
	}

	var employees []models.EmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&employees); err != nil {
		t.Fatalf("failed to decode page 1 response: %v", err)
	}

	if len(employees) != 5 {
		t.Errorf("expected 5 employees on page 1, got %d", len(employees))
	}

	// Ensure that the password field is not exposed.
	for _, emp := range employees {
		if emp.Password != "" {
			t.Errorf("password field should not be exposed for employee %s", emp.Email)
		}
	}

	// Now test page=2, size=5.
	getURL = testServer.URL + "/employees?page=2&size=5"
	resp, err = http.Get(getURL)
	if err != nil {
		t.Fatalf("GET request failed for page 2: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for page 2, got %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&employees); err != nil {
		t.Fatalf("failed to decode page 2 response: %v", err)
	}

	if len(employees) != 5 {
		t.Errorf("expected 5 employees on page 2, got %d", len(employees))
	}

	// Again check that the password field is not exposed.
	for _, emp := range employees {
		if emp.Password != "" {
			t.Errorf("password field should not be exposed for employee %s", emp.Email)
		}
	}

	t.Log("TestE2E_ListEmployees_Pagination passed")
}
func TestE2E_CreateEmployee_InvalidEmail(t *testing.T) {
	// Create an employee with an invalid email (missing '@').
	newEmployee := models.Employee{
		Email: "invalidemail", // invalid format
		Name:  "Invalid Email User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, err := json.Marshal(newEmployee)
	if err != nil {
		t.Fatalf("failed to marshal employee: %v", err)
	}
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		// Optionally, log the response body for debugging.
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400 for invalid email, got %d; response: %s", resp.StatusCode, string(respBody))
	} else {
		t.Log("TestE2E_CreateEmployee_InvalidEmail passed")
	}
}
func TestE2E_CreateEmployee_DuplicateEmail(t *testing.T) {
	// Create a new employee payload.
	duplicateEmployee := models.Employee{
		Email: "duplicate@example.com",
		Name:  "Duplicate Email User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, err := json.Marshal(duplicateEmployee)
	if err != nil {
		t.Fatalf("failed to marshal employee: %v", err)
	}
	// First attempt: should succeed.
	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request for first employee: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for first employee, got %d", resp.StatusCode)
	}

	// Second attempt: should fail with conflict.
	resp2, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request for duplicate employee: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409 for duplicate email, got %d", resp2.StatusCode)
	} else {
		t.Log("TestE2E_CreateEmployee_DuplicateEmail passed")
	}
}

// TestE2E_ListEmployees_ByEmailDomain tests GET /employees?criteria=byEmailDomain&value={domain}&page={page}&size={size}
func TestE2E_ListEmployees_ByEmailDomain(t *testing.T) {
	// Create employees with different email domains.
	employees := []models.Employee{
		{
			Email: "alice@other1.com",
			Name:  "Alice",
			Birthdate: models.Birthdate{
				Day:   "01",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Developer"},
			Password: "Test1",
		},
		{
			Email: "bob@other1.com",
			Name:  "Bob",
			Birthdate: models.Birthdate{
				Day:   "02",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Developer"},
			Password: "Test1",
		},
		{
			Email: "charlie@other.com",
			Name:  "Charlie",
			Birthdate: models.Birthdate{
				Day:   "03",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Developer"},
			Password: "Test1",
		},
	}

	// Insert all employees.
	for _, emp := range employees {
		body, _ := json.Marshal(emp)
		resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("failed to create employee %s: %v", emp.Email, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to create employee %s, status: %d", emp.Email, resp.StatusCode)
		}
	}

	// Query employees with domain "example.com"
	getURL := fmt.Sprintf("%s/employees?criteria=byEmailDomain&value=other1.com&page=1&size=10", testServer.URL)
	resp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to GET employees by email domain: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var results []models.EmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Expect exactly 2 employees (alice and bob).
	if len(results) != 2 {
		t.Errorf("expected 2 employees for domain 'example.com', got %d", len(results))
	}

	// Verify that none of the returned employees expose the password.
	for _, emp := range results {
		if emp.Password != "" {
			t.Errorf("password field should be omitted for employee %s", emp.Email)
		}
	}

	t.Log("TestE2E_ListEmployees_ByEmailDomain passed")
}

// TestE2E_ListEmployees_ByRole tests GET /employees?criteria=byRole&value={role}&page={page}&size={size}
func TestE2E_ListEmployees_ByRole(t *testing.T) {
	// Create employees with different roles.
	employees := []models.Employee{
		{
			Email: "dave@example.com",
			Name:  "Dave",
			Birthdate: models.Birthdate{
				Day:   "04",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Manager"},
			Password: "Test1",
		},
		{
			Email: "eve@example.com",
			Name:  "Eve",
			Birthdate: models.Birthdate{
				Day:   "05",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Developer"},
			Password: "Test1",
		},
		{
			Email: "frank@example.com",
			Name:  "Frank",
			Birthdate: models.Birthdate{
				Day:   "06",
				Month: "01",
				Year:  "1990",
			},
			Roles:    []string{"Manager"},
			Password: "Test1",
		},
	}

	// Insert all employees.
	for _, emp := range employees {
		body, _ := json.Marshal(emp)
		resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("failed to create employee %s: %v", emp.Email, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to create employee %s, status: %d", emp.Email, resp.StatusCode)
		}
	}

	// Query employees with role "Manager"
	getURL := fmt.Sprintf("%s/employees?criteria=byRole&value=Manager&page=1&size=10", testServer.URL)
	resp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to GET employees by role: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var results []models.EmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Expect exactly 2 employees with role Manager (Dave and Frank).
	if len(results) != 2 {
		t.Errorf("expected 2 employees for role 'Manager', got %d", len(results))
	}

	// Verify that none of the returned employees expose the password.
	for _, emp := range results {
		if emp.Password != "" {
			t.Errorf("password field should be omitted for employee %s", emp.Email)
		}
	}

	t.Log("TestE2E_ListEmployees_ByRole passed")
}

func TestE2E_ListEmployees_ByAge(t *testing.T) {
	// Get current time.
	now := time.Now()

	// --- Create Employee: Exactly 30 years old ---
	// We choose January 1 so that the birthday has already passed this year.
	emp30 := models.Employee{
		Email: "age30@example.com",
		Name:  "Age 30 User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  fmt.Sprintf("%d", now.Year()-30),
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}
	body30, err := json.Marshal(emp30)
	if err != nil {
		t.Fatalf("failed to marshal employee age 30: %v", err)
	}
	resp30, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body30))
	if err != nil {
		t.Fatalf("failed to create employee age 30: %v", err)
	}
	defer resp30.Body.Close()
	if resp30.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee age 30, got %d", resp30.StatusCode)
	}

	// --- Create Employee: 29 years and 364 days old ---
	// To simulate an employee who is one day shy of turning 30,
	// we set the birthday to tomorrow with a birth year such that the computed age is 29.
	tomorrow := now.Add(24 * time.Hour)
	emp29 := models.Employee{
		Email: "age29@example.com",
		Name:  "Age 29 User",
		Birthdate: models.Birthdate{
			Day:   fmt.Sprintf("%02d", tomorrow.Day()),
			Month: fmt.Sprintf("%02d", int(tomorrow.Month())),
			Year:  fmt.Sprintf("%d", now.Year()-30),
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}
	body29, err := json.Marshal(emp29)
	if err != nil {
		t.Fatalf("failed to marshal employee age 29: %v", err)
	}
	resp29, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body29))
	if err != nil {
		t.Fatalf("failed to create employee age 29: %v", err)
	}
	defer resp29.Body.Close()
	if resp29.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee age 29, got %d", resp29.StatusCode)
	}

	// --- Create Employee: Exactly 31 years old ---
	emp31 := models.Employee{
		Email: "age31@example.com",
		Name:  "Age 31 User",
		Birthdate: models.Birthdate{
			Day:   "01",
			Month: "01",
			Year:  fmt.Sprintf("%d", now.Year()-31),
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}
	body31, err := json.Marshal(emp31)
	if err != nil {
		t.Fatalf("failed to marshal employee age 31: %v", err)
	}
	resp31, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body31))
	if err != nil {
		t.Fatalf("failed to create employee age 31: %v", err)
	}
	defer resp31.Body.Close()
	if resp31.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee age 31, got %d", resp31.StatusCode)
	}

	// --- Query employees by age 30 ---
	getURL := fmt.Sprintf("%s/employees?criteria=byAge&value=%d&page=1&size=10", testServer.URL, 30)
	resp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to GET employees by age 30: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for age 30 search, got %d", resp.StatusCode)
	}

	var results []models.EmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response for age 30 search: %v", err)
	}

	// Expect only the exactly 30-year-old employee to appear.
	if len(results) != 1 {
		t.Errorf("expected exactly one employee of age 30, got %d", len(results))
	}

	// Verify that the returned employee is the 30-year-old and does not expose the password.
	for _, emp := range results {
		if emp.Email != "age30@example.com" {
			t.Errorf("unexpected employee %s returned in age 30 search", emp.Email)
		}
		if emp.Password != "" {
			t.Errorf("password field should be omitted for employee %s", emp.Email)
		}
	}

	t.Log("TestE2E_ListEmployees_ByAge passed: only the employee exactly 30 years old is returned")
}
func TestE2E_CreateEmployee_FutureBirthdate(t *testing.T) {
	// Calculate a future birthdate (e.g., tomorrow's date).
	futureDate := time.Now().Add(24 * time.Hour)
	// Format day, month, and year with zero padding if needed.
	day := fmt.Sprintf("%02d", futureDate.Day())
	month := fmt.Sprintf("%02d", int(futureDate.Month()))
	year := fmt.Sprintf("%d", futureDate.Year())

	newEmployee := models.Employee{
		Email: "futurebirthday@example.com",
		Name:  "Future Birthday User",
		Birthdate: models.Birthdate{
			Day:   day,
			Month: month,
			Year:  year,
		},
		Roles:    []string{"Developer"},
		Manager:  nil,
		Password: "Test1",
	}
	body, err := json.Marshal(newEmployee)
	if err != nil {
		t.Fatalf("failed to marshal employee: %v", err)
	}

	resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// We expect the API to reject a future birthdate (HTTP 400 Bad Request).
	if resp.StatusCode != http.StatusBadRequest {
		// Optionally log the response body for debugging.
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400 for future birthdate, got %d; response: %s", resp.StatusCode, string(respBody))
	} else {
		t.Log("TestE2E_CreateEmployee_FutureBirthdate passed")
	}
}

// TestE2E_SetAndGetManager tests setting a manager for an employee and retrieving it.
func TestE2E_SetAndGetManager(t *testing.T) {
	// First, create an employee and a manager.
	employee := models.Employee{
		Email: "employeeM1@example.com",
		Name:  "Employee One",
		Birthdate: models.Birthdate{
			Day:   "10",
			Month: "05",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}
	manager := models.Employee{
		Email: "manager1@example.com",
		Name:  "Manager One",
		Birthdate: models.Birthdate{
			Day:   "05",
			Month: "03",
			Year:  "1985",
		},
		Roles:    []string{"Manager"},
		Password: "Test1",
	}

	// Create employee.
	bodyEmp, _ := json.Marshal(employee)
	respEmp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyEmp))
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	respEmp.Body.Close()
	if respEmp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee creation, got %d", respEmp.StatusCode)
	}

	// Create manager.
	bodyMgr, _ := json.Marshal(manager)
	respMgr, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyMgr))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	respMgr.Body.Close()
	if respMgr.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for manager creation, got %d", respMgr.StatusCode)
	}

	// Now, set the manager for the employee.
	managerBoundary := map[string]string{"email": manager.Email}
	bodyBoundary, _ := json.Marshal(managerBoundary)
	putURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	req, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBuffer(bodyBoundary))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	putResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send PUT request: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for setting manager, got %d", putResp.StatusCode)
	}

	// Retrieve the manager for the employee.
	getURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for getting manager, got %d", getResp.StatusCode)
	}

	var mgrResp models.EmployeeResponse
	if err := json.NewDecoder(getResp.Body).Decode(&mgrResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	if mgrResp.Email != manager.Email {
		t.Errorf("expected manager email %s, got %s", manager.Email, mgrResp.Email)
	}
	t.Log("TestE2E_SetAndGetManager passed")
}

// TestE2E_GetSubordinates tests retrieving subordinates for a manager.
func TestE2E_GetSubordinates(t *testing.T) {
	// Create a manager.
	manager := models.Employee{
		Email: "manager2@example.com",
		Name:  "Manager Two",
		Birthdate: models.Birthdate{
			Day:   "07",
			Month: "04",
			Year:  "1980",
		},
		Roles:    []string{"Manager"},
		Password: "Test1",
	}
	bodyMgr, _ := json.Marshal(manager)
	respMgr, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyMgr))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	respMgr.Body.Close()
	if respMgr.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for manager creation, got %d", respMgr.StatusCode)
	}

	// Create two employees and set their manager to the above manager.
	subordinateEmails := []string{"sub1@example.com", "sub2@example.com"}
	for _, email := range subordinateEmails {
		emp := models.Employee{
			Email: email,
			Name:  "Subordinate " + email,
			Birthdate: models.Birthdate{
				Day:   "12",
				Month: "06",
				Year:  "1992",
			},
			Roles:    []string{"Developer"},
			Password: "Test1",
		}
		bodyEmp, _ := json.Marshal(emp)
		resp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyEmp))
		if err != nil {
			t.Fatalf("failed to create subordinate %s: %v", email, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for subordinate creation, got %d", resp.StatusCode)
		}

		// Set manager for subordinate.
		managerBoundary := map[string]string{"email": manager.Email}
		bodyBoundary, _ := json.Marshal(managerBoundary)
		putURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, email)
		req, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBuffer(bodyBoundary))
		if err != nil {
			t.Fatalf("failed to create PUT request for subordinate %s: %v", email, err)
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		putResp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to send PUT request for subordinate %s: %v", email, err)
		}
		putResp.Body.Close()
		if putResp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for setting manager for subordinate %s, got %d", email, putResp.StatusCode)
		}
	}

	// Now, get subordinates for the manager using pagination (page=1, size=10).
	getURL := fmt.Sprintf("%s/employees/%s/subordinates?page=1&size=10", testServer.URL, manager.Email)
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request for subordinates: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for getting subordinates, got %d", getResp.StatusCode)
	}

	var subs []models.EmployeeResponse
	if err := json.NewDecoder(getResp.Body).Decode(&subs); err != nil {
		t.Fatalf("failed to decode subordinates response: %v", err)
	}

	if len(subs) != len(subordinateEmails) {
		t.Errorf("expected %d subordinates, got %d", len(subordinateEmails), len(subs))
	}
	// Check that password fields are not exposed.
	for _, emp := range subs {
		if emp.Password != "" {
			t.Errorf("password should not be exposed for subordinate %s", emp.Email)
		}
	}
	t.Log("TestE2E_GetSubordinates passed")
}

// TestE2E_DeleteManager tests disconnecting the manager relationship.
func TestE2E_DeleteManager(t *testing.T) {
	// Create an employee and a manager, then set the manager relationship.
	employee := models.Employee{
		Email: "employeeM2@example.com",
		Name:  "Employee Two",
		Birthdate: models.Birthdate{
			Day:   "15",
			Month: "07",
			Year:  "1991",
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}
	manager := models.Employee{
		Email: "manager3@example.com",
		Name:  "Manager Three",
		Birthdate: models.Birthdate{
			Day:   "20",
			Month: "08",
			Year:  "1982",
		},
		Roles:    []string{"Manager"},
		Password: "Test1",
	}

	// Create employee.
	bodyEmp, _ := json.Marshal(employee)
	respEmp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyEmp))
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	respEmp.Body.Close()
	if respEmp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee creation, got %d", respEmp.StatusCode)
	}

	// Create manager.
	bodyMgr, _ := json.Marshal(manager)
	respMgr, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyMgr))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	respMgr.Body.Close()
	if respMgr.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for manager creation, got %d", respMgr.StatusCode)
	}

	// Set the manager for the employee.
	managerBoundary := map[string]string{"email": manager.Email}
	bodyBoundary, _ := json.Marshal(managerBoundary)
	putURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	req, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBuffer(bodyBoundary))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	putResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send PUT request: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for setting manager, got %d", putResp.StatusCode)
	}

	// Now, delete the manager relationship.
	delURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	delReq, err := http.NewRequest(http.MethodDelete, delURL, nil)
	if err != nil {
		t.Fatalf("failed to create DELETE request: %v", err)
	}
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("failed to send DELETE request: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for deleting manager, got %d", delResp.StatusCode)
	}

	// Finally, try to GET the manager for the employee; expect an error (e.g. 404).
	getURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request after deletion: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 status after manager deletion, got %d", getResp.StatusCode)
	} else {
		t.Log("TestE2E_DeleteManager passed")
	}
}

// TestE2E_DeleteAllEmployees tests that DELETE /employees clears all employee data,
// including any relationships (like manager settings).
func TestE2E_DeleteAllEmployees(t *testing.T) {
	// Create a new employee.
	employee := models.Employee{
		Email: "deleteTestEmployee@example.com",
		Name:  "Delete Test Employee",
		Birthdate: models.Birthdate{
			Day:   "10",
			Month: "05",
			Year:  "1990",
		},
		Roles:    []string{"Developer"},
		Password: "Test1",
	}

	// Create a manager.
	manager := models.Employee{
		Email: "deleteTestManager@example.com",
		Name:  "Delete Test Manager",
		Birthdate: models.Birthdate{
			Day:   "05",
			Month: "03",
			Year:  "1985",
		},
		Roles:    []string{"Manager"},
		Password: "Test1",
	}

	// Create employee.
	bodyEmp, _ := json.Marshal(employee)
	respEmp, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyEmp))
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	respEmp.Body.Close()
	if respEmp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for employee creation, got %d", respEmp.StatusCode)
	}

	// Create manager.
	bodyMgr, _ := json.Marshal(manager)
	respMgr, err := http.Post(testServer.URL+"/employees", "application/json", bytes.NewBuffer(bodyMgr))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	respMgr.Body.Close()
	if respMgr.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for manager creation, got %d", respMgr.StatusCode)
	}

	// Set the manager for the employee.
	managerBoundary := map[string]string{"email": manager.Email}
	bodyBoundary, _ := json.Marshal(managerBoundary)
	putURL := fmt.Sprintf("%s/employees/%s/manager", testServer.URL, employee.Email)
	req, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBuffer(bodyBoundary))
	if err != nil {
		t.Fatalf("failed to create PUT request for setting manager: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	putResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send PUT request for setting manager: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for setting manager, got %d", putResp.StatusCode)
	}

	// Now delete all employees by sending DELETE to /employees.
	delReq, err := http.NewRequest(http.MethodDelete, testServer.URL+"/employees", nil)
	if err != nil {
		t.Fatalf("failed to create DELETE request: %v", err)
	}
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("failed to send DELETE request: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for DELETE /employees, got %d", delResp.StatusCode)
	}

	// Try to GET the employee; expecting a 404 (or not found error).
	getURL := fmt.Sprintf("%s/employees/%s?password=%s", testServer.URL, employee.Email, employee.Password)
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("failed to send GET request after DELETE: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after deleting all employees, got %d", getResp.StatusCode)
	} else {
		t.Log("TestE2E_DeleteAllEmployees passed: employee no longer exists")
	}
}
