package router

import (
	"WebMVCEmployees/controllers"
	_ "WebMVCEmployees/docs"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter initializes the Gin router with API routes and Swagger UI.
func SetupRouter(empController *controllers.EmployeeController) *gin.Engine {
	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	employeeRoutes := r.Group("/employees")
	{
		employeeRoutes.POST("", empController.CreateEmployeeHandler)
		employeeRoutes.DELETE("", empController.DeleteAllEmployeesHandler)
		employeeRoutes.PUT("/:employeeEmail/manager", empController.SetManagerHandler)
		employeeRoutes.GET("/:employeeEmail/manager", empController.GetManagerHandler)
		employeeRoutes.DELETE("/:employeeEmail/manager", empController.RemoveManagerHandler)
		employeeRoutes.GET("/:employeeEmail/subordinates", empController.GetSubordinatesHandler)
		employeeRoutes.GET("/:employeeEmail", empController.GetEmployeeHandler)

		// Separate filtering endpoints.
		employeeRoutes.GET("", empController.ListEmployeesHandler)
	}

	return r
}

// SetupServer creates and returns an HTTP server configured with your router.
func SetupServer(empController *controllers.EmployeeController) *http.Server {
	router := SetupRouter(empController)
	return &http.Server{
		Addr:    ":8080", // You can parameterize this if needed.
		Handler: router,
	}
}
