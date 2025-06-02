package handlers

import (
	"bytes"
	"encoding/json"
	"errors" // Added for store error simulation
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/gaanon/gorecipes_v2/models"
	"github.com/gaanon/gorecipes_v2/store/mocks" // Import the generated mocks
)

// Helper function to create a new Gin engine for testing
func setupTestRouter(handler *RecipeHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	api := router.Group("/api/v1")
	{
		api.POST("/recipes", handler.CreateRecipe)
		// Add other routes as you test them
	}
	return router
}

func TestRecipeHandler_CreateRecipe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeReq := &models.RecipeRequest{
		Title:           "Test Recipe",
		Description:     strPtr("Test Description"),
		Serves:          intPtr(4),
		PrepTimeMinutes: intPtr(15),
		CookTimeMinutes: intPtr(30),
		Ingredients: []models.RecipeIngredientRequest{
			{IngredientName: "Test Ingredient", Quantity: float64Ptr(1.0), UnitName: strPtr("cup"), SortOrder: 1},
		},
		Steps: []models.RecipeStepRequest{
			{StepNumber: 1, Instruction: "Test Step"},
		},
		Tags: []models.RecipeTagRequest{
			{Name: "Test Tag"},
		},
	}

	createdRecipeID := uuid.New()
	expectedRecipe := &models.Recipe{
		ID:              createdRecipeID,
		Title:           recipeReq.Title,
		Description:     recipeReq.Description, // recipeReq.Description is *string, expectedRecipe.Description is *string
		Serves:          recipeReq.Serves,          // recipeReq.Serves is *int, expectedRecipe.Serves is *int
		PrepTimeMinutes: recipeReq.PrepTimeMinutes, // recipeReq.PrepTimeMinutes is *int, expectedRecipe.PrepTimeMinutes is *int
		CookTimeMinutes: recipeReq.CookTimeMinutes, // recipeReq.CookTimeMinutes is *int, expectedRecipe.CookTimeMinutes is *int
		CreatedAt:       time.Now(),                // Actual time will be set by DB, this is for comparison structure
		UpdatedAt:       time.Now(),                // Actual time will be set by DB, this is for comparison structure
	}

	// Calculate TotalTimeMinutes carefully, considering nil pointers
	var totalTimeCalc int
	if recipeReq.PrepTimeMinutes != nil {
		totalTimeCalc += *recipeReq.PrepTimeMinutes
	}
	if recipeReq.CookTimeMinutes != nil {
		totalTimeCalc += *recipeReq.CookTimeMinutes
	}
	if recipeReq.PrepTimeMinutes != nil || recipeReq.CookTimeMinutes != nil {
		expectedRecipe.TotalTimeMinutes = &totalTimeCalc
	} else {
		expectedRecipe.TotalTimeMinutes = nil // Explicitly nil if both inputs are nil
	}

	mockStore.EXPECT().CreateRecipe(gomock.Any(), recipeReq).Return(expectedRecipe, nil).Times(1)

	jsonBody, _ := json.Marshal(recipeReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/recipes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseRecipe models.Recipe
	err := json.Unmarshal(w.Body.Bytes(), &responseRecipe)
	assert.NoError(t, err)
	assert.Equal(t, expectedRecipe.Title, responseRecipe.Title)
	assert.Equal(t, expectedRecipe.ID, responseRecipe.ID)
}

func TestRecipeHandler_CreateRecipe_BindError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	// Invalid JSON
	jsonBody := []byte("{\"title\": \"Test Recipe\", \"description\": \"Test Description\"invalidjson")
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/recipes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Check for specific error message if desired
}

func TestRecipeHandler_CreateRecipe_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	// Request missing required 'Title'
	recipeReq := &models.RecipeRequest{
		Description: strPtr("Test Description"),
	}

	jsonBody, _ := json.Marshal(recipeReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/recipes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Check for specific validation error details
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["details"].(map[string]interface{}), "Title")
}

// Helper functions for pointers to make test setup cleaner
func strPtr(s string) *string       { return &s }
func intPtr(i int) *int             { return &i }
func float64Ptr(f float64) *float64 { return &f }

func TestRecipeHandler_CreateRecipe_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeReq := &models.RecipeRequest{
		Title:           "Test Recipe Store Error",
		Description:     strPtr("Description for store error test"),
		Serves:          intPtr(2),
		PrepTimeMinutes: intPtr(10),
		CookTimeMinutes: intPtr(20),
		Ingredients: []models.RecipeIngredientRequest{
			{IngredientName: "Ingredient", Quantity: float64Ptr(1.0), UnitName: strPtr("each"), SortOrder: 1},
		},
		Steps: []models.RecipeStepRequest{
			{StepNumber: 1, Instruction: "A step"},
		},
	}

	// Simulate a store error
	expectedStoreErrorMessage := "database unavailable"
	mockStore.EXPECT().CreateRecipe(gomock.Any(), recipeReq).Return(nil, errors.New(expectedStoreErrorMessage)).Times(1)

	jsonBody, _ := json.Marshal(recipeReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/recipes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	
	assert.NotNil(t, errorResponse["error"])
	errorMessage, ok := errorResponse["error"].(string)
	assert.True(t, ok, "Error message should be a string")
	assert.Contains(t, errorMessage, "Failed to create recipe")
	assert.Contains(t, errorMessage, expectedStoreErrorMessage)

	assert.NotNil(t, errorResponse["status"])
	status, ok := errorResponse["status"].(float64) // JSON numbers are float64
	assert.True(t, ok, "Status should be a number")
	assert.Equal(t, float64(http.StatusInternalServerError), status)
}


