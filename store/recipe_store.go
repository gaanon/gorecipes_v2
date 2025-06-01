package store

import (
	"context"
	"errors" // Added for pgx.ErrNoRows check
	"fmt"

	"github.com/gaanon/gorecipes_v2/models" // Adjust import path if needed
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecipeStore defines the interface for recipe data operations.
type RecipeStore interface {
	CreateRecipe(ctx context.Context, recipeReq *models.RecipeRequest) (*models.Recipe, error)
	GetRecipeByID(ctx context.Context, id uuid.UUID) (*models.Recipe, error)
	ListRecipes(ctx context.Context) ([]*models.Recipe, error) // Simplified for now, add filters/pagination later
	UpdateRecipe(ctx context.Context, id uuid.UUID, recipeReq *models.RecipeRequest) (*models.Recipe, error)
	DeleteRecipe(ctx context.Context, id uuid.UUID) error
}

// findOrCreateIngredient finds an ingredient by name or creates it if not found.
func findOrCreateIngredient(ctx context.Context, tx pgx.Tx, ingredientName string) (uuid.UUID, error) {
	var ingredientID uuid.UUID
	// Try to find existing ingredient
	err := tx.QueryRow(ctx, "SELECT id FROM ingredients WHERE name = $1", ingredientName).Scan(&ingredientID)
	if err == nil {
		return ingredientID, nil // Found
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("failed to query ingredient by name %s: %w", ingredientName, err)
	}

	// Not found, create new ingredient (DB defaults created_at)
	ingredientID = uuid.New()
	_, err = tx.Exec(ctx, "INSERT INTO ingredients (id, name) VALUES ($1, $2)", ingredientID, ingredientName)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create new ingredient %s: %w", ingredientName, err)
	}
	return ingredientID, nil
}

// findOrCreateMeasurementUnit finds a measurement unit by name or creates it if not found.
// For newly created units, it uses 'other' as the default system and NULL for other optional fields.
func findOrCreateMeasurementUnit(ctx context.Context, tx pgx.Tx, unitName string) (uuid.UUID, error) {
	var unitID uuid.UUID
	err := tx.QueryRow(ctx, "SELECT id FROM measurement_units WHERE name = $1", unitName).Scan(&unitID)
	if err == nil {
		return unitID, nil // Found
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("failed to query measurement unit by name %s: %w", unitName, err)
	}

	// Not found, create new measurement unit
	unitID = uuid.New()
	// DB defaults created_at; system is required, others are optional. Defaulting to 'metric'.
	_, err = tx.Exec(ctx, "INSERT INTO measurement_units (id, name, system) VALUES ($1, $2, $3)", unitID, unitName, "metric")
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create new measurement unit %s: %w", unitName, err)
	}
	return unitID, nil
}

// findOrCreateTag finds a tag by name or creates it if not found.
func findOrCreateTag(ctx context.Context, tx pgx.Tx, tagName string) (uuid.UUID, error) {
	var tagID uuid.UUID
	err := tx.QueryRow(ctx, "SELECT id FROM tags WHERE name = $1", tagName).Scan(&tagID)
	if err == nil {
		return tagID, nil // Found
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("failed to query tag by name %s: %w", tagName, err)
	}

	// Not found, create new tag (DB defaults created_at)
	tagID = uuid.New()
	_, err = tx.Exec(ctx, "INSERT INTO tags (id, name) VALUES ($1, $2)", tagID, tagName)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create new tag %s: %w", tagName, err)
	}
	return tagID, nil
}

// DBRecipeStore implements the RecipeStore interface using a pgxpool.Pool.
type DBRecipeStore struct {
	db *pgxpool.Pool
}

// NewRecipeStore creates a new DBRecipeStore.
func NewRecipeStore(db *pgxpool.Pool) *DBRecipeStore {
	return &DBRecipeStore{db: db}
}

// CreateRecipe inserts a new recipe and its associated data (ingredients, steps, tags) into the database.
// This operation is performed within a single transaction.
// It returns the fully populated Recipe object.
func (s *DBRecipeStore) CreateRecipe(ctx context.Context, recipeReq *models.RecipeRequest) (*models.Recipe, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if commit is not called

	newRecipeID := uuid.New()
	recipeSQL := `
		INSERT INTO recipes (id, title, description, photo_filename, serves, prep_time_minutes, cook_time_minutes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;`
	var createdRecipeID uuid.UUID
	err = tx.QueryRow(ctx, recipeSQL,
		newRecipeID,
		recipeReq.Title,
		recipeReq.Description,
		recipeReq.PhotoFilename,
		recipeReq.Serves,
		recipeReq.PrepTimeMinutes,
		recipeReq.CookTimeMinutes,
		recipeReq.CreatedBy,
	).Scan(&createdRecipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe: %w", err)
	}

	// Insert ingredients with find-or-create logic
	for _, ingReq := range recipeReq.Ingredients { // ingReq is models.RecipeIngredientRequest
		ingredientID, err := findOrCreateIngredient(ctx, tx, ingReq.IngredientName)
		if err != nil {
			return nil, fmt.Errorf("processing ingredient %s: %w", ingReq.IngredientName, err)
		}

		var unitIDPtr *uuid.UUID // Use a pointer to handle potential NULL unit_id
		if ingReq.UnitName != nil && *ingReq.UnitName != "" {
			foundUnitID, err := findOrCreateMeasurementUnit(ctx, tx, *ingReq.UnitName)
			if err != nil {
				return nil, fmt.Errorf("processing measurement unit %s: %w", *ingReq.UnitName, err)
			}
			unitIDPtr = &foundUnitID
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit_id, notes, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6);`,
			createdRecipeID, ingredientID, ingReq.Quantity, unitIDPtr, ingReq.Notes, ingReq.SortOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to insert recipe ingredient link for %s: %w", ingReq.IngredientName, err)
		}
	}

	// Insert steps
	for _, stepReq := range recipeReq.Steps {
		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_steps (recipe_id, step_number, instruction, duration_minutes, temperature)
			VALUES ($1, $2, $3, $4, $5);`,
			createdRecipeID, stepReq.StepNumber, stepReq.Instruction, stepReq.DurationMinutes, stepReq.Temperature)
		if err != nil {
			return nil, fmt.Errorf("failed to insert recipe step %d: %w", stepReq.StepNumber, err)
		}
	}

	// Insert tags with find-or-create logic
	for _, tagReq := range recipeReq.Tags { // tagReq is models.RecipeTagRequest
		tagID, err := findOrCreateTag(ctx, tx, tagReq.Name)
		if err != nil {
			return nil, fmt.Errorf("processing tag %s: %w", tagReq.Name, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_tags (recipe_id, tag_id)
			VALUES ($1, $2);`,
			createdRecipeID, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert recipe tag link for %s: %w", tagReq.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return the fully populated recipe
	return s.GetRecipeByID(ctx, createdRecipeID)
}

// GetRecipeByID retrieves a single recipe by its ID, including its ingredients, steps, and tags.
func (s *DBRecipeStore) GetRecipeByID(ctx context.Context, id uuid.UUID) (*models.Recipe, error) {
	recipe := &models.Recipe{}

	// 1. Get main recipe details
	recipeSQL := `
		SELECT r.id, r.title, r.description, r.photo_filename, r.serves, 
		       r.prep_time_minutes, r.cook_time_minutes, r.total_time_minutes, 
		       r.created_at, r.updated_at, r.created_by
		FROM recipes r
		WHERE r.id = $1;`
	err := s.db.QueryRow(ctx, recipeSQL, id).Scan(
		&recipe.ID, &recipe.Title, &recipe.Description, &recipe.PhotoFilename, &recipe.Serves,
		&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
		&recipe.CreatedAt, &recipe.UpdatedAt, &recipe.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("recipe with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get recipe %s: %w", id, err)
	}

	// 2. Get recipe ingredients
	ingredientsSQL := `
		SELECT 
			ri.ingredient_id, 
			i.name AS ingredient_name, 
			i.category AS ingredient_description, 
			ri.quantity, 
			ri.notes, 
			ri.sort_order,
			mu.id AS unit_id,             -- This will be scanned into tempUnit.ID (*uuid.UUID)
			mu.name AS unit_name,           -- This will be scanned into tempUnit.Name (string)
			mu.abbreviation AS unit_abbreviation, -- This will be scanned into tempUnit.Abbreviation (*string)
			mu.system AS unit_system         -- This will be scanned into tempUnit.System (models.MeasurementSystem)
		FROM recipe_ingredients ri
		JOIN ingredients i ON ri.ingredient_id = i.id
		LEFT JOIN measurement_units mu ON ri.unit_id = mu.id
		WHERE ri.recipe_id = $1
		ORDER BY ri.sort_order;`
	rows, err := s.db.Query(ctx, ingredientsSQL, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingredients for recipe %s: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var ing models.RecipeIngredient
		var tempUnit models.MeasurementUnit // Temporary unit to scan into, ID is now *uuid.UUID
		err := rows.Scan(
			&ing.IngredientID,
			&ing.IngredientName,        // Scans i.name
			&ing.IngredientDescription, // Scans i.description
			&ing.Quantity,
			&ing.Notes,
			&ing.SortOrder,
			&tempUnit.ID,               // Scans mu.id (which can be NULL)
			&tempUnit.Name,             // Scans mu.name
			&tempUnit.Abbreviation,     // Scans mu.abbreviation
			&tempUnit.System,           // Scans mu.system
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ingredient for recipe %s: %w", id, err)
		}

		// If unit_id was not null (tempUnit.ID is not nil), assign the scanned unit to the ingredient
		if tempUnit.ID != nil {
			ing.Unit = &tempUnit
		} else {
		    // Ensure ing.Unit is nil if there's no unit information from the DB
		    ing.Unit = nil 
		}
		recipe.Ingredients = append(recipe.Ingredients, ing)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating ingredients for recipe %s: %w", id, rows.Err())
	}

	// 3. Get recipe steps
	stepsSQL := `
		SELECT step_number, instruction, duration_minutes, temperature
		FROM recipe_steps
		WHERE recipe_id = $1
		ORDER BY step_number;`
	rows, err = s.db.Query(ctx, stepsSQL, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps for recipe %s: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var step models.RecipeStep
		err := rows.Scan(&step.StepNumber, &step.Instruction, &step.DurationMinutes, &step.Temperature)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step for recipe %s: %w", id, err)
		}
		recipe.Steps = append(recipe.Steps, step)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating steps for recipe %s: %w", id, rows.Err())
	}

	// 4. Get recipe tags
	tagsSQL := `
		SELECT t.id, t.name
		FROM recipe_tags rt
		JOIN tags t ON rt.tag_id = t.id
		WHERE rt.recipe_id = $1
		ORDER BY t.name;`
	rows, err = s.db.Query(ctx, tagsSQL, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for recipe %s: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(&tag.ID, &tag.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag for recipe %s: %w", id, err)
		}
		recipe.Tags = append(recipe.Tags, tag)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating tags for recipe %s: %w", id, rows.Err())
	}

	return recipe, nil
}

// ListRecipes retrieves a list of all recipes with their basic details.
// TODO: Implement pagination and filtering.
func (s *DBRecipeStore) ListRecipes(ctx context.Context) ([]*models.Recipe, error) {
	listSQL := `
		SELECT id, title, description, photo_filename, serves, 
		       prep_time_minutes, cook_time_minutes, total_time_minutes, 
		       created_at, updated_at, created_by
		FROM recipes
		ORDER BY updated_at DESC; -- Or by title, created_at, etc.
	`
	rows, err := s.db.Query(ctx, listSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}
	defer rows.Close()

	var recipes []*models.Recipe
	for rows.Next() {
		recipe := &models.Recipe{}
		err := rows.Scan(
			&recipe.ID, &recipe.Title, &recipe.Description, &recipe.PhotoFilename, &recipe.Serves,
			&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
			&recipe.CreatedAt, &recipe.UpdatedAt, &recipe.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recipe during list: %w", err)
		}
		recipes = append(recipes, recipe)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating recipes list: %w", rows.Err())
	}

	return recipes, nil
}

// UpdateRecipe updates an existing recipe and its associated data.
// It replaces ingredients, steps, and tags rather than performing a diff.
// This operation is performed within a single transaction.
func (s *DBRecipeStore) UpdateRecipe(ctx context.Context, id uuid.UUID, recipeReq *models.RecipeRequest) (*models.Recipe, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction for update: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Update the main recipe details in 'recipes' table
	updateRecipeSQL := `
		UPDATE recipes
		SET title = $2, description = $3, photo_filename = $4, serves = $5, 
		    prep_time_minutes = $6, cook_time_minutes = $7, created_by = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id; -- Check if the recipe existed
	`
	var updatedRecipeID uuid.UUID
	err = tx.QueryRow(ctx, updateRecipeSQL,
		id,
		recipeReq.Title,
		recipeReq.Description,
		recipeReq.PhotoFilename,
		recipeReq.Serves,
		recipeReq.PrepTimeMinutes,
		recipeReq.CookTimeMinutes,
		recipeReq.CreatedBy,
	).Scan(&updatedRecipeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("recipe with ID %s not found for update", id)
		}
		return nil, fmt.Errorf("failed to update recipe %s: %w", id, err)
	}

	// 2. Delete existing associated data (ingredients, steps, tags)
	// ON DELETE CASCADE in DB schema might handle some of this, but explicit deletion is safer for junction tables if not all relations are cascaded.
	// For recipe_ingredients, recipe_steps, recipe_tags, ON DELETE CASCADE is on recipe_id.
	// So, we just need to delete from these tables before re-inserting.

	deleteRelationsSQL := []string{
		`DELETE FROM recipe_ingredients WHERE recipe_id = $1;`,
		`DELETE FROM recipe_steps WHERE recipe_id = $1;`,
		`DELETE FROM recipe_tags WHERE recipe_id = $1;`,
	}

	for _, sql := range deleteRelationsSQL {
		_, err = tx.Exec(ctx, sql, id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete old relations for recipe %s: %w", id, err)
		}
	}

	// 3. Insert new associated data (ingredients, steps, tags) - similar to CreateRecipe
	// Insert ingredients with find-or-create logic
	for _, ingReq := range recipeReq.Ingredients { // ingReq is models.RecipeIngredientRequest
		ingredientID, err := findOrCreateIngredient(ctx, tx, ingReq.IngredientName)
		if err != nil {
			return nil, fmt.Errorf("processing ingredient %s for update: %w", ingReq.IngredientName, err)
		}

		var unitIDPtr *uuid.UUID
		if ingReq.UnitName != nil && *ingReq.UnitName != "" {
			foundUnitID, err := findOrCreateMeasurementUnit(ctx, tx, *ingReq.UnitName)
			if err != nil {
				return nil, fmt.Errorf("processing measurement unit %s for update: %w", *ingReq.UnitName, err)
			}
			unitIDPtr = &foundUnitID
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit_id, notes, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6);`,
			id, ingredientID, ingReq.Quantity, unitIDPtr, ingReq.Notes, ingReq.SortOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to insert updated recipe ingredient link for %s: %w", ingReq.IngredientName, err)
		}
	}

	// Insert steps
	for _, stepReq := range recipeReq.Steps {
		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_steps (recipe_id, step_number, instruction, duration_minutes, temperature)
			VALUES ($1, $2, $3, $4, $5);`,
			id, stepReq.StepNumber, stepReq.Instruction, stepReq.DurationMinutes, stepReq.Temperature)
		if err != nil {
			return nil, fmt.Errorf("failed to insert updated recipe step %d: %w", stepReq.StepNumber, err)
		}
	}

	// Insert tags with find-or-create logic
	for _, tagReq := range recipeReq.Tags { // tagReq is models.RecipeTagRequest
		tagID, err := findOrCreateTag(ctx, tx, tagReq.Name)
		if err != nil {
			return nil, fmt.Errorf("processing tag %s for update: %w", tagReq.Name, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_tags (recipe_id, tag_id)
			VALUES ($1, $2);`,
			id, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert updated recipe tag link for %s: %w", tagReq.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit update transaction for recipe %s: %w", id, err)
	}

	return s.GetRecipeByID(ctx, id) // Return the updated, fully populated recipe
}

// DeleteRecipe removes a recipe from the database by its ID.
// Associated data in recipe_ingredients, recipe_steps, and recipe_tags
// should be deleted automatically due to ON DELETE CASCADE constraints on the recipe_id foreign key.
func (s *DBRecipeStore) DeleteRecipe(ctx context.Context, id uuid.UUID) error {
	deleteSQL := `DELETE FROM recipes WHERE id = $1`

	cmdTag, err := s.db.Exec(ctx, deleteSQL, id)
	if err != nil {
		return fmt.Errorf("failed to delete recipe with ID %s: %w", id, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("recipe with ID %s not found for deletion", id) // Or a specific error type like ErrNotFound
	}

	return nil
}

// Implement other RecipeStore methods (GetRecipeByID, ListRecipes, UpdateRecipe, DeleteRecipe) here...
