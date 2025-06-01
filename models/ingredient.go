package models

import (
	"time"

	"github.com/google/uuid"
)

// Ingredient represents an ingredient item.
type Ingredient struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Category  *string   `json:"category,omitempty" db:"category"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MeasurementUnit represents a unit of measurement.
type MeasurementUnit struct {
	ID               *uuid.UUID        `json:"id,omitempty" db:"id"` // Made pointer to handle NULL from LEFT JOIN
	Name             *string           `json:"name,omitempty" db:"name"` // Changed to pointer to handle NULL
	Abbreviation     *string           `json:"abbreviation,omitempty" db:"abbreviation"`
	System           *MeasurementSystem `json:"system,omitempty" db:"system"` // From common.go; Changed to pointer
	BaseUnitID       *uuid.UUID        `json:"base_unit_id,omitempty" db:"base_unit_id"`
	ConversionFactor *float64          `json:"conversion_factor,omitempty" db:"conversion_factor"`
}

// RecipeIngredient links a Recipe to an Ingredient with quantity and unit details.
type RecipeIngredient struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	RecipeID     uuid.UUID  `json:"-" db:"recipe_id"` // Often omitted from JSON if part of a Recipe struct
	IngredientID uuid.UUID  `json:"ingredient_id" db:"ingredient_id"`
	Quantity     *float64   `json:"quantity,omitempty" db:"quantity"`
	UnitID       *uuid.UUID `json:"unit_id,omitempty" db:"unit_id"`
	Notes        *string    `json:"notes,omitempty" db:"notes"`
	SortOrder    int        `json:"sort_order" db:"sort_order"`

	// Fields to populate from related tables for richer API responses
	IngredientName        *string            `json:"ingredient_name,omitempty"`        // From Ingredient table
	IngredientDescription *string            `json:"ingredient_description,omitempty"` // From Ingredient table
	Unit                  *MeasurementUnit   `json:"unit,omitempty"`                   // Populated from MeasurementUnit table
}

// RecipeIngredientRequest is used when creating/updating recipe ingredients.
// It might reference an existing ingredient by ID or allow creating a new one (more complex, for now by ID).
type RecipeIngredientRequest struct {
	IngredientName string     `json:"ingredient_name" validate:"required"`
	Quantity       *float64   `json:"quantity" validate:"omitempty,gt=0"`
	UnitName       *string    `json:"unit_name" validate:"omitempty"` // e.g., "grams", "ml", "cup"; backend will find or create
	Notes          *string    `json:"notes"`
	SortOrder      int        `json:"sort_order" validate:"gte=0"`
}
