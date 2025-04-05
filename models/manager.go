package models

// ManagerEmailBoundary is used to bind manager email in bonus endpoints.
// swagger:model
type ManagerEmailBoundary struct {
	// The email of the manager.
	Email string `json:"email" example:"manager@s.example.com"`
}