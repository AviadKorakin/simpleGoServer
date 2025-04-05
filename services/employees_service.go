package services

import (
	"context"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"time"

	"WebMVCEmployees/errors"
	"WebMVCEmployees/models"
	"WebMVCEmployees/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EmployeeService provides business logic for managing employees.
type EmployeeService struct {
	Repo *repository.EmployeeRepository
}

// NewEmployeeService creates a new EmployeeService using the provided repository.
func NewEmployeeService(repo *repository.EmployeeRepository) *EmployeeService {
	return &EmployeeService{
		Repo: repo,
	}
}

func (s *EmployeeService) CreateEmployee(ctx context.Context, emp models.Employee) (models.Employee, error) {
	// Basic validations:
	if emp.Email == "" || emp.Name == "" {
		return models.Employee{}, errors.NewHTTPError(http.StatusBadRequest, "email and name are required")
	}
	if err := validateEmail(emp.Email); err != nil {
		return models.Employee{}, errors.NewHTTPError(http.StatusBadRequest, "invalid email format")
	}

	// Validate birthdate using the separate helper function.
	if err := validateBirthdate(emp.Birthdate); err != nil {
		return models.Employee{}, err
	}
	// Validate password using the helper function.
	if err := validatePassword(emp.Password); err != nil {
		return models.Employee{}, err
	}
	if emp.Manager != nil {
		if err := s.validateManager(ctx, *emp.Manager); err != nil {
			return models.Employee{}, err
		}
	}
	// Insert the new employee into MongoDB.
	_, err := s.Repo.Collection.InsertOne(ctx, emp)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return models.Employee{}, errors.NewHTTPError(http.StatusConflict, "employee with this email already exists")
		}
		return models.Employee{}, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Remove the password before returning the response.
	emp.Password = ""
	return emp, nil
}

// validateEmail checks if the provided email is valid.
func validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}

// validateBirthdate checks that the birthdate fields are of correct length and numeric.
func validateBirthdate(birthdate models.Birthdate) error {
	// Check lengths.
	if len(birthdate.Day) != 2 {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate day must be two digits")
	}
	if len(birthdate.Month) != 2 {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate month must be two digits")
	}
	if len(birthdate.Year) != 4 {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate year must be four digits")
	}

	// Convert to integers.
	day, err := strconv.Atoi(birthdate.Day)
	if err != nil {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate day must be numeric")
	}
	month, err := strconv.Atoi(birthdate.Month)
	if err != nil {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate month must be numeric")
	}
	year, err := strconv.Atoi(birthdate.Year)
	if err != nil {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate year must be numeric")
	}

	// Create a time.Time object from the birthdate.
	birthDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// Ensure the birthdate is not in the future.
	if birthDate.After(time.Now().UTC()) {
		return errors.NewHTTPError(http.StatusBadRequest, "birthdate cannot be in the future")
	}

	return nil
}
func validatePassword(password string) error {
	if len(password) < 3 {
		return errors.NewHTTPError(http.StatusBadRequest, "password must be at least 3 characters")
	}

	hasDigit := false
	hasUpper := false
	for _, ch := range password {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		}
		if ch >= 'A' && ch <= 'Z' {
			hasUpper = true
		}
	}
	if !hasDigit || !hasUpper {
		return errors.NewHTTPError(http.StatusBadRequest, "password must contain at least one digit and one uppercase letter")
	}
	return nil
}

// ValidateManager checks if the manager with the given email exists.
func (s *EmployeeService) validateManager(ctx context.Context, managerEmail string) error {
	if managerEmail == "" {
		return nil // No manager to validate
	}
	var manager models.Employee
	err := s.Repo.Collection.FindOne(ctx, bson.M{models.EmployeeRef.Email: managerEmail}).Decode(&manager)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.NewHTTPError(http.StatusBadRequest, "manager not found")
		}
		return errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// GetEmployee retrieves an employee by email and password.
// It returns an error if no matching employee is found.
func (s *EmployeeService) GetEmployee(ctx context.Context, email, password string) (models.Employee, error) {
	var emp models.Employee
	filter := bson.M{models.EmployeeRef.Email: email, models.EmployeeRef.Password: password}
	err := s.Repo.Collection.FindOne(ctx, filter).Decode(&emp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Employee{}, errors.NewHTTPError(http.StatusNotFound, "employee not found")
		}
		return models.Employee{}, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Do not expose the password in the response.
	emp.Password = ""
	return emp, nil
}

// GetAllEmployees returns all employees with pagination.
func (s *EmployeeService) GetAllEmployees(ctx context.Context, page, size int) ([]models.Employee, error) {
	skip := int64((page - 1) * size)
	limit := int64(size)
	findOptions := options.Find().SetSort(bson.D{{Key: models.EmployeeRef.Email, Value: 1}}).SetSkip(skip).SetLimit(limit)
	cursor, err := s.Repo.Collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer cursor.Close(ctx)

	var employees []models.Employee
	if err = cursor.All(ctx, &employees); err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Ensure employees is not nil.
	if employees == nil {
		employees = []models.Employee{}
	}
	// Remove passwords from all employee responses.
	for i := range employees {
		employees[i].Password = ""
	}
	return employees, nil
}

// GetEmployeesByEmailDomain returns employees whose email domain matches exactly.
func (s *EmployeeService) GetEmployeesByEmailDomain(ctx context.Context, domain string, page, size int) ([]models.Employee, error) {
	filter := bson.M{models.EmployeeRef.Email: bson.M{"$regex": "@" + domain + "$", "$options": "i"}}
	skip := int64((page - 1) * size)
	limit := int64(size)
	findOptions := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.Repo.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer cursor.Close(ctx)

	var employees []models.Employee
	if err = cursor.All(ctx, &employees); err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Ensure employees is not nil.
	if employees == nil {
		employees = []models.Employee{}
	}

	for i := range employees {
		employees[i].Password = ""
	}
	return employees, nil
}

// GetEmployeesByRole returns employees having a specific role.
func (s *EmployeeService) GetEmployeesByRole(ctx context.Context, role string, page, size int) ([]models.Employee, error) {
	filter := bson.M{models.EmployeeRef.Roles: role}
	skip := int64((page - 1) * size)
	limit := int64(size)
	findOptions := options.Find().SetSort(bson.D{{Key: models.EmployeeRef.Email, Value: 1}}).SetSkip(skip).SetLimit(limit)
	cursor, err := s.Repo.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer cursor.Close(ctx)

	var employees []models.Employee
	if err = cursor.All(ctx, &employees); err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Ensure employees is not nil.
	if employees == nil {
		employees = []models.Employee{}
	}
	for i := range employees {
		employees[i].Password = ""
	}
	return employees, nil
}

// GetEmployeesByAge returns employees whose age in years equals the specified value.
// Assumes that the current date is provided as a Unix timestamp.
func (s *EmployeeService) GetEmployeesByAge(ctx context.Context, ageInYears int, currentUnix int64, page, size int) ([]models.Employee, error) {
	cursor, err := s.Repo.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer cursor.Close(ctx)

	var employees []models.Employee
	if err = cursor.All(ctx, &employees); err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var filtered []models.Employee
	now := time.Unix(currentUnix, 0)
	for _, emp := range employees {
		bYear, err := strconv.Atoi(emp.Birthdate.Year)
		if err != nil {
			continue
		}
		bMonth, err := strconv.Atoi(emp.Birthdate.Month)
		if err != nil {
			continue
		}
		bDay, err := strconv.Atoi(emp.Birthdate.Day)
		if err != nil {
			continue
		}
		birthDate := time.Date(bYear, time.Month(bMonth), bDay, 0, 0, 0, 0, time.UTC)
		calculatedAge := now.Year() - birthDate.Year()
		if now.YearDay() < birthDate.YearDay() {
			calculatedAge--
		}
		if calculatedAge == ageInYears {
			emp.Password = ""
			filtered = append(filtered, emp)
		}
	}

	// Sort filtered employees by birth date (ascending order)
	sort.Slice(filtered, func(i, j int) bool {
		// Convert the birthdates to time.Time for comparison.
		byear, _ := strconv.Atoi(filtered[i].Birthdate.Year)
		bmonth, _ := strconv.Atoi(filtered[i].Birthdate.Month)
		bday, _ := strconv.Atoi(filtered[i].Birthdate.Day)
		dateI := time.Date(byear, time.Month(bmonth), bday, 0, 0, 0, 0, time.UTC)

		byearJ, _ := strconv.Atoi(filtered[j].Birthdate.Year)
		bmonthJ, _ := strconv.Atoi(filtered[j].Birthdate.Month)
		bdayJ, _ := strconv.Atoi(filtered[j].Birthdate.Day)
		dateJ := time.Date(byearJ, time.Month(bmonthJ), bdayJ, 0, 0, 0, 0, time.UTC)

		return dateI.Before(dateJ)
	})

	// Ensure filtered is an empty slice (not nil) if no records found.
	if filtered == nil {
		filtered = []models.Employee{}
	}

	// Apply pagination to the filtered slice.
	start := (page - 1) * size
	if start > len(filtered) {
		return []models.Employee{}, nil
	}
	end := start + size
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// DeleteAllEmployees deletes all employee documents from the collection.
func (s *EmployeeService) DeleteAllEmployees(ctx context.Context) error {
	_, err := s.Repo.Collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// Bonus: Manager relationship endpoints

// SetManager sets or updates the manager for an employee.
func (s *EmployeeService) SetManager(ctx context.Context, employeeEmail string, managerEmail string) error {
	var emp models.Employee
	err := s.Repo.Collection.FindOne(ctx, bson.M{models.EmployeeRef.Email: employeeEmail}).Decode(&emp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.NewHTTPError(http.StatusNotFound, "employee not found")
		}
		return errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := s.validateManager(ctx, managerEmail); err != nil {
		return err
	}
	_, err = s.Repo.Collection.UpdateOne(ctx, bson.M{models.EmployeeRef.Email: employeeEmail},
		bson.M{"$set": bson.M{models.EmployeeRef.Manager: managerEmail}})
	if err != nil {
		return errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// GetManager retrieves the manager for a given employee.
func (s *EmployeeService) GetManager(ctx context.Context, employeeEmail string) (models.Employee, error) {
	var emp models.Employee
	err := s.Repo.Collection.FindOne(ctx, bson.M{models.EmployeeRef.Email: employeeEmail}).Decode(&emp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Employee{}, errors.NewHTTPError(http.StatusNotFound, "employee not found")
		}
		return models.Employee{}, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if emp.Manager == nil {
		return models.Employee{}, errors.NewHTTPError(http.StatusNotFound, "manager not set")
	}
	var manager models.Employee
	err = s.Repo.Collection.FindOne(ctx, bson.M{models.EmployeeRef.Email: *emp.Manager}).Decode(&manager)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Employee{}, errors.NewHTTPError(http.StatusNotFound, "manager not found")
		}
		return models.Employee{}, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	manager.Password = ""
	return manager, nil
}

// GetSubordinates returns employees managed by the given managerEmail, with pagination.
func (s *EmployeeService) GetSubordinates(ctx context.Context, managerEmail string, page, size int) ([]models.Employee, error) {
	filter := bson.M{models.EmployeeRef.Manager: managerEmail}
	skip := int64((page - 1) * size)
	limit := int64(size)
	findOptions := options.Find().SetSort(bson.D{{Key: models.EmployeeRef.Email, Value: 1}}).SetSkip(skip).SetLimit(limit)
	cursor, err := s.Repo.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer cursor.Close(ctx)
	var subordinates []models.Employee
	if err = cursor.All(ctx, &subordinates); err != nil {
		return nil, errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for i := range subordinates {
		subordinates[i].Password = ""
	}
	return subordinates, nil
}

// RemoveManager unsets the manager for an employee.
func (s *EmployeeService) RemoveManager(ctx context.Context, employeeEmail string) error {
	_, err := s.Repo.Collection.UpdateOne(ctx, bson.M{models.EmployeeRef.Email: employeeEmail},
		bson.M{"$unset": bson.M{models.EmployeeRef.Manager: ""}})
	if err != nil {
		return errors.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}
