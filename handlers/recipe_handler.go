package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gaanon/gorecipes_v2/models"
	"github.com/gaanon/gorecipes_v2/store"
)

// Global validator instance
var validate = validator.New()

// RecipeHandler handles HTTP requests for recipes.
type RecipeHandler struct {
	store store.RecipeStore
}

// NewRecipeHandler creates a new RecipeHandler.
func NewRecipeHandler(store store.RecipeStore) *RecipeHandler {
	return &RecipeHandler{store: store}
}

// CreateRecipe handles the creation of a new recipe.
// @Summary Create a new recipe
// @Description Create a new recipe with ingredients, steps, and tags.
// @Tags recipes
// @Accept json
// @Produce json
// @Param recipe body models.RecipeRequest true "Recipe to create"
// @Success 201 {object} models.Recipe
// @Failure 400 {object} APIError "Invalid input"
// @Failure 500 {object} APIError "Server error"
// @Router /recipes [post]
func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var req models.RecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Validate the request
	if err := validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		RespondWithDetailedError(c, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	recipe, err := h.store.CreateRecipe(c.Request.Context(), &req)
	if err != nil {
		// More specific error handling can be added here based on error types from store
		RespondWithError(c, http.StatusInternalServerError, "Failed to create recipe: "+err.Error())
		return
	}
	RespondWithJSON(c, http.StatusCreated, recipe)
}

// GetRecipe handles fetching a single recipe by its ID.
// @Summary Get a recipe by ID
// @Description Get a single recipe by its UUID, including ingredients, steps, and tags.
// @Tags recipes
// @Produce json
// @Param id path string true "Recipe ID (UUID)"
// @Success 200 {object} models.Recipe
// @Failure 400 {object} APIError "Invalid ID format"
// @Failure 404 {object} APIError "Recipe not found"
// @Failure 500 {object} APIError "Server error"
// @Router /recipes/{id} [get]
func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	idStr := c.Param("id")
	recipeID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid recipe ID format: "+err.Error())
		return
	}

	recipe, err := h.store.GetRecipeByID(c.Request.Context(), recipeID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") { // Basic check, improve with custom errors
			RespondWithError(c, http.StatusNotFound, "Recipe not found: "+err.Error())
		} else {
			RespondWithError(c, http.StatusInternalServerError, "Failed to get recipe: "+err.Error())
		}
		return
	}
	RespondWithJSON(c, http.StatusOK, recipe)
}

// ListRecipes handles fetching a list of recipes.
// @Summary List recipes
// @Description Get a list of all recipes (basic details).
// @Tags recipes
// @Produce json
// @Success 200 {array} models.Recipe
// @Failure 500 {object} APIError "Server error"
// @Router /recipes [get]
func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	// TODO: Add pagination and filtering query parameters
	recipes, err := h.store.ListRecipes(c.Request.Context())
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "Failed to list recipes: "+err.Error())
		return
	}
	RespondWithJSON(c, http.StatusOK, recipes)
}

// UpdateRecipe handles updating an existing recipe.
// @Summary Update an existing recipe
// @Description Update an existing recipe by its UUID. All fields are replaced.
// @Tags recipes
// @Accept json
// @Produce json
// @Param id path string true "Recipe ID (UUID)"
// @Param recipe body models.RecipeRequest true "Recipe data to update"
// @Success 200 {object} models.Recipe
// @Failure 400 {object} APIError "Invalid input or ID format"
// @Failure 404 {object} APIError "Recipe not found"
// @Failure 500 {object} APIError "Server error"
// @Router /recipes/{id} [put]
// formatValidationErrors converts validator.ValidationErrors into a map for a structured JSON response.
func formatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrs {
			// fieldErr.Namespace() gives full path e.g., "RecipeRequest.Ingredients[0].IngredientName"
			// We strip the top-level struct name for a cleaner field key.
			namespace := fieldErr.Namespace()
			parts := strings.SplitN(namespace, ".", 2)
			fieldName := namespace // Default to full namespace if not in expected format (e.g. top-level field)
			if len(parts) > 1 {
				fieldName = parts[1]
			}
			errors[fieldName] = fmt.Sprintf("failed on '%s' validation (value: '%v')", fieldErr.Tag(), fieldErr.Value())
		}
	}
	return errors
}

// RespondWithDetailedError sends a JSON error response with additional details.
func RespondWithDetailedError(c *gin.Context, code int, message string, details interface{}) {
	c.JSON(code, gin.H{"status": code, "error": message, "details": details})
}

func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	idStr := c.Param("id")
	recipeID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid recipe ID format: "+err.Error())
		return
	}

	var req models.RecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Validate the request
	if err := validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		RespondWithDetailedError(c, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	recipe, err := h.store.UpdateRecipe(c.Request.Context(), recipeID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") { // Basic check
			RespondWithError(c, http.StatusNotFound, "Recipe not found for update: "+err.Error())
		} else {
			RespondWithError(c, http.StatusInternalServerError, "Failed to update recipe: "+err.Error())
		}
		return
	}
	RespondWithJSON(c, http.StatusOK, recipe)
}

// DeleteRecipe handles deleting a recipe by its ID.
// @Summary Delete a recipe by ID
// @Description Delete a single recipe by its UUID.
// @Tags recipes
// @Produce json
// @Param id path string true "Recipe ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} APIError "Invalid ID format"
// @Failure 404 {object} APIError "Recipe not found"
// @Failure 500 {object} APIError "Server error"
// @Router /recipes/{id} [delete]
func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	idStr := c.Param("id")
	recipeID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid recipe ID format: "+err.Error())
		return
	}

	err = h.store.DeleteRecipe(c.Request.Context(), recipeID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") { // Basic check
			RespondWithError(c, http.StatusNotFound, "Recipe not found for deletion: "+err.Error())
		} else {
			RespondWithError(c, http.StatusInternalServerError, "Failed to delete recipe: "+err.Error())
		}
		return
	}
	RespondWithJSON(c, http.StatusNoContent, nil) // Or c.Status(http.StatusNoContent)
}
