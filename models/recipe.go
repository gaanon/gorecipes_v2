package models

import (
	"time"

	"github.com/google/uuid"
)

// Recipe represents a cooking recipe.
// Note: total_time_minutes and search_vector are generated columns in the DB.
// total_time_minutes is included here as it's useful data to return.
// search_vector is typically handled at the DB query level and not directly in the model for CRUD.
type Recipe struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Title            string     `json:"title" db:"title"`
	Description      *string    `json:"description,omitempty" db:"description"`
	PhotoFilename    *string    `json:"photo_filename,omitempty" db:"photo_filename"`
	Serves           *int       `json:"serves,omitempty" db:"serves"`
	PrepTimeMinutes  *int       `json:"prep_time_minutes,omitempty" db:"prep_time_minutes"`
	CookTimeMinutes  *int       `json:"cook_time_minutes,omitempty" db:"cook_time_minutes"`
	TotalTimeMinutes *int       `json:"total_time_minutes,omitempty" db:"total_time_minutes"` // Read-only from DB
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty" db:"created_by"`

	// Fields for related data, to be populated when fetching a full recipe
	Ingredients []RecipeIngredient `json:"ingredients,omitempty"`
	Steps       []RecipeStep       `json:"steps,omitempty"`
	Tags        []Tag              `json:"tags,omitempty"`
}

// RecipeRequest is used for creating or updating a recipe.
// It might omit fields like ID, CreatedAt, UpdatedAt, TotalTimeMinutes which are auto-generated or set by the server.
// It also allows for more specific validation if needed.
type RecipeRequest struct {
	Title           string             `json:"title" validate:"required,min=3,max=255"`
	Description     *string            `json:"description"`
	PhotoFilename   *string            `json:"photo_filename" validate:"omitempty,max=255"`
	Serves          *int               `json:"serves" validate:"omitempty,gt=0"`
	PrepTimeMinutes *int               `json:"prep_time_minutes" validate:"omitempty,gte=0"`
	CookTimeMinutes *int               `json:"cook_time_minutes" validate:"omitempty,gte=0"`
	CreatedBy       *uuid.UUID         `json:"created_by"` // Optional, depends on auth context

	Ingredients []RecipeIngredientRequest `json:"ingredients" validate:"omitempty,dive"`
	Steps       []RecipeStepRequest       `json:"steps" validate:"omitempty,dive"`
	Tags        []RecipeTagRequest        `json:"tags" validate:"omitempty,dive"`      // For creating/associating tags by name
}
