package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"WebMVCEmployees/errors"
	"WebMVCEmployees/models"
	"WebMVCEmployees/services"

	"github.com/gin-gonic/gin"
)

// EmployeeController handles HTTP requests for employee resources.
type EmployeeController struct {
	Service *services.EmployeeService
}

// NewEmployeeController creates a new EmployeeController.
func NewEmployeeController(s *services.EmployeeService) *EmployeeController {
	return &EmployeeController{
		Service: s,
	}
}

// CreateEmployeeHandler handles POST /employees
// @Summary Create a new employee
// @Description Accepts employee details in JSON, validates and stores the employee.
// @Tags employees
// @Accept json
// @Produce json
// @Param employee body models.Employee true "Employee details"
// @Success 200 {object} models.EmployeeResponse
// @Router /employees [post]
func (c *EmployeeController) CreateEmployeeHandler(ctx *gin.Context) {
	var emp models.Employee
	if err := ctx.ShouldBindJSON(&emp); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	createdEmp, err := c.Service.CreateEmployee(cx, emp)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	ctx.JSON(http.StatusOK, createdEmp)
}

// GetEmployeeHandler handles GET /employees/{employeeEmail}?password={password}
// @Summary Get an employee by email and password
// @Description Returns employee details if the provided email and password match a record.
// @Tags employees
// @Produce json
// @Param employeeEmail path string true "Employee email"
// @Param password query string true "Employee password"
// @Success 200 {object} models.EmployeeResponse
// @Router /employees/{employeeEmail} [get]
func (c *EmployeeController) GetEmployeeHandler(ctx *gin.Context) {
	email := ctx.Param("employeeEmail")
	password := ctx.Query("password")
	if email == "" || password == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing email or password"})
		return
	}

	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	emp, err := c.Service.GetEmployee(cx, email, password)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	ctx.JSON(http.StatusOK, emp)
}

// ListEmployeesHandler handles GET /employees with filtering and pagination.
// @Summary List employees with filtering
// @Description Returns a paginated list of employees. When the "criteria" query parameter is provided,
// it filters employees by email domain, role, or age. If no employees match the criteria, an empty array is returned.
// Passwords are not exposed.
// @Tags employees
// @Produce json
// @Param criteria query string false "Filter criteria. Allowed values: byEmailDomain,byRole,byAge. If set to 'none' or omitted, all employees are returned" Enums(byEmailDomain,byRole,byAge) default()
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Success 200 {array} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse "Bad Request"
// @Router /employees [get]
func (c *EmployeeController) ListEmployeesHandler(ctx *gin.Context) {
	// Parse pagination parameters.
	page, err := strconv.Atoi(ctx.Query("page"))
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil || size < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size parameter"})
		return
	}
	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	criteria := ctx.Query("criteria")
	var employees []models.Employee
	switch criteria {
	case "byEmailDomain":
		domain := ctx.Query("value")
		if domain == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing domain value"})
			return
		}
		employees, err = c.listEmployeesByEmailDomain(cx, domain, page, size)
	case "byRole":
		role := ctx.Query("value")
		if role == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing role value"})
			return
		}
		employees, err = c.listEmployeesByRole(cx, role, page, size)
	case "byAge":
		ageStr := ctx.Query("value")
		age, errConv := strconv.Atoi(ageStr)
		if errConv != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid age value"})
			return
		}
		employees, err = c.listEmployeesByAge(cx, age, page, size)
	default:
		employees, err = c.Service.GetAllEmployees(cx, page, size)
	}
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, employees)
}

// Private helper methods to reuse service logic for filtering.
func (c *EmployeeController) listEmployeesByEmailDomain(cx context.Context, domain string, page, size int) ([]models.Employee, error) {
	return c.Service.GetEmployeesByEmailDomain(cx, domain, page, size)
}

func (c *EmployeeController) listEmployeesByRole(cx context.Context, role string, page, size int) ([]models.Employee, error) {
	return c.Service.GetEmployeesByRole(cx, role, page, size)
}

func (c *EmployeeController) listEmployeesByAge(cx context.Context, age int, page, size int) ([]models.Employee, error) {
	// Use current Unix time for age calculation.
	return c.Service.GetEmployeesByAge(cx, age, time.Now().Unix(), page, size)
}

// handleError is a helper function to process errors.
func handleError(ctx *gin.Context, err error) {
	if httpErr, ok := err.(*errors.HTTPError); ok {
		ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
	} else {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

// DeleteAllEmployeesHandler handles DELETE /employees
// @Summary Delete all employees
// @Description Deletes all employee records from the service.
// @Tags employees
// @Produce json
// @Success 200 {object} map[string]string "Success message"
// @Failure 500 {object} models.ErrorResponse
// @Router /employees [delete]
func (c *EmployeeController) DeleteAllEmployeesHandler(ctx *gin.Context) {
	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	err := c.Service.DeleteAllEmployees(cx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "All employees deleted"})
}

// SetManagerHandler handles PUT /employees/{employeeEmail}/manager
// @Summary Set manager for an employee
// @Description Associates an employee with a manager using ManagerEmailBoundary JSON.
// @Tags employees
// @Accept json
// @Produce json
// @Param employeeEmail path string true "Employee email"
// @Param manager body models.ManagerEmailBoundary true "Manager email"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /employees/{employeeEmail}/manager [put]
func (c *EmployeeController) SetManagerHandler(ctx *gin.Context) {
	employeeEmail := ctx.Param("employeeEmail")
	var mb models.ManagerEmailBoundary
	if err := ctx.ShouldBindJSON(&mb); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	if err := c.Service.SetManager(cx, employeeEmail, mb.Email); err != nil {
		// You may check for "not found" error.
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Manager set successfully"})
}

// GetManagerHandler handles GET /employees/{employeeEmail}/manager
// @Summary Get manager of an employee
// @Description Returns the manager details (excluding password) for the specified employee.
// @Tags employees
// @Produce json
// @Param employeeEmail path string true "Employee email"
// @Success 200 {object} models.EmployeeResponse
// @Failure 404 {object} models.ErrorResponse "Not Found"
// @Router /employees/{employeeEmail}/manager [get]
func (c *EmployeeController) GetManagerHandler(ctx *gin.Context) {
	employeeEmail := ctx.Param("employeeEmail")
	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	manager, err := c.Service.GetManager(cx, employeeEmail)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	ctx.JSON(http.StatusOK, manager)
}

// GetSubordinatesHandler handles GET /managers/{managerEmail}/subordinates?page={page}&size={size}
// @Summary Get subordinates for a manager
// @Description Returns a paginated list of employees managed by the specified manager.
// @Tags employees
// @Produce json
// @Param managerEmail path string true "Manager email"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Success 200 {array} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse "Bad Request"
// @Router /managers/{managerEmail}/subordinates [get]
func (c *EmployeeController) GetSubordinatesHandler(ctx *gin.Context) {
	managerEmail := ctx.Param("employeeEmail")
	page, err := strconv.Atoi(ctx.Query("page"))
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil || size < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size parameter"})
		return
	}
	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	subordinates, err := c.Service.GetSubordinates(cx, managerEmail, page, size)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	ctx.JSON(http.StatusOK, subordinates)
}

// RemoveManagerHandler handles DELETE /employees/{employeeEmail}/manager
// @Summary Remove manager association from an employee
// @Description Unsets the manager for the specified employee.
// @Tags employees
// @Produce json
// @Param employeeEmail path string true "Employee email"
// @Success 200 {object} map[string]string "Success message"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /employees/{employeeEmail}/manager [delete]
func (c *EmployeeController) RemoveManagerHandler(ctx *gin.Context) {
	employeeEmail := ctx.Param("employeeEmail")
	cx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	if err := c.Service.RemoveManager(cx, employeeEmail); err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			ctx.JSON(httpErr.Code, gin.H{"error": httpErr.Msg})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Manager removed successfully"})
}
