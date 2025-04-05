package models

// FieldNames groups together the field names for an Employee.
type FieldNames struct {
	Email     string
	Name      string
	Password  string
	Birthdate string
	Roles     string
	Manager   string
}

// EmployeeFields is an instance containing the field names.
var EmployeeRef = FieldNames{
	Email:     "email",
	Name:      "name",
	Password:  "password",
	Birthdate: "birthdate",
	Roles:     "roles",
	Manager:   "manager",
}

// Birthdate represents an employee's date of birth.
// swagger:model Birthdate
type Birthdate struct {
	// Day represents the two-digit day.
	Day string `json:"day" example:"03"`
	// Month represents the two-digit month.
	Month string `json:"month" example:"01"`
	// Year represents the four-digit year.
	Year string `json:"year" example:"1999"`
}

// Employee represents an employee record.
// swagger:model Employee
// @Description An employee with email, name, password, birthdate, and roles.
type Employee struct {
	// Email is the unique identifier.
	Email string `json:"email" example:"janesmith@s.afeka.ac.il"`
	// Name is the full name of the employee.
	Name string `json:"name" example:"Jane Smith"`
	// Password is the employee's password. It is omitted in responses.
	Password string `json:"password,omitempty" example:"Pa5"`
	// Birthdate contains the employee's date of birth.
	Birthdate Birthdate `json:"birthdate"`
	// Roles contains the roles or permissions of the employee.
	Roles []string `json:"roles" example:"DevOps,R&D"`
	// Manager optionally stores the email of the employee's manager.
	Manager *string `json:"manager,omitempty" example:"manager@s.example.com"`
}

// Employee represents an employee record.
// swagger:model EmployeeResponse
// @Description An employee with email, name, password, birthdate, and roles.
type EmployeeResponse struct {
	// Email is the unique identifier.
	Email string `json:"email" example:"janesmith@s.afeka.ac.il"`
	// Name is the full name of the employee.
	Name string `json:"name" example:"Jane Smith"`
	// Password is the employee's password. It is omitted in responses.
	Password string `json:"-"`
	// Birthdate contains the employee's date of birth.
	Birthdate Birthdate `json:"birthdate"`
	// Roles contains the roles or permissions of the employee.
	Roles []string `json:"roles" example:"DevOps,R&D"`
	// Manager optionally stores the email of the employee's manager.
	Manager *string `json:"manager,omitempty" example:"manager@s.example.com"`
}
