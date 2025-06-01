package models

import (
	"time"

	"github.com/google/uuid"
)

// RecipeStep represents a single step in a recipe's instructions.
type RecipeStep struct {
	ID              uuid.UUID `json:"id" db:"id"`
	RecipeID        uuid.UUID `json:"-" db:"recipe_id"` // Often omitted from JSON if part of a Recipe struct
	StepNumber      int       `json:"step_number" db:"step_number"`
	Instruction     string    `json:"instruction" db:"instruction"`
	DurationMinutes *int      `json:"duration_minutes,omitempty" db:"duration_minutes"`
	Temperature     *string   `json:"temperature,omitempty" db:"temperature"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// RecipeStepRequest is used when creating/updating recipe steps.
type RecipeStepRequest struct {
	StepNumber      int     `json:"step_number" validate:"required,gte=1"`
	Instruction     string  `json:"instruction" validate:"required,min=1"`
	DurationMinutes *int    `json:"duration_minutes" validate:"omitempty,gte=0"`
	Temperature     *string `json:"temperature" validate:"omitempty,max=50"`
}
