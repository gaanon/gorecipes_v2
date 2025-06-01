package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag represents a category or keyword for a recipe.
type Tag struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Color       *string   `json:"color,omitempty" db:"color"` // Hex color code
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Note: The RecipeTag junction table is handled by associating TagIDs in RecipeRequest
// and potentially embedding a slice of Tag structs in the Recipe model for responses.
// A separate RecipeTag struct might be used internally in the store layer if needed.

// RecipeTagRequest is used when creating/updating recipe tags by name.
type RecipeTagRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}
