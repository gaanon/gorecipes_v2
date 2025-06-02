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
		api.PUT("/recipes/:id", handler.UpdateRecipe)
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

func TestRecipeHandler_UpdateRecipe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeID := uuid.New()
	recipeReq := &models.RecipeRequest{
		Title:           "Updated Test Recipe",
		Description:     strPtr("Updated Test Description"),
		Serves:          intPtr(5),
		PrepTimeMinutes: intPtr(20),
		CookTimeMinutes: intPtr(35),
		Ingredients: []models.RecipeIngredientRequest{
			{IngredientName: "Updated Ingredient", Quantity: float64Ptr(2.0), UnitName: strPtr("tbsp"), SortOrder: 1},
		},
		Steps: []models.RecipeStepRequest{
			{StepNumber: 1, Instruction: "Updated Test Step"},
		},
		Tags: []models.RecipeTagRequest{
			{Name: "Updated Test Tag"},
		},
	}

	expectedRecipe := &models.Recipe{
		ID:              recipeID,
		Title:           recipeReq.Title,
		Description:     recipeReq.Description,
		Serves:          recipeReq.Serves,
		PrepTimeMinutes: recipeReq.PrepTimeMinutes,
		CookTimeMinutes: recipeReq.CookTimeMinutes,
		UpdatedAt:       time.Now(), // Store will set this, mock for comparison
	}
	// Calculate TotalTimeMinutes
	var totalTimeCalc int
	if recipeReq.PrepTimeMinutes != nil {
		totalTimeCalc += *recipeReq.PrepTimeMinutes
	}
	if recipeReq.CookTimeMinutes != nil {
		totalTimeCalc += *recipeReq.CookTimeMinutes
	}
	if recipeReq.PrepTimeMinutes != nil || recipeReq.CookTimeMinutes != nil {
		expectedRecipe.TotalTimeMinutes = &totalTimeCalc
	}


	mockStore.EXPECT().UpdateRecipe(gomock.Any(), recipeID, recipeReq).Return(expectedRecipe, nil).Times(1)

	jsonBody, _ := json.Marshal(recipeReq)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/"+recipeID.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseRecipe models.Recipe
	err := json.Unmarshal(w.Body.Bytes(), &responseRecipe)
	assert.NoError(t, err)
	assert.Equal(t, expectedRecipe.Title, responseRecipe.Title)
	assert.Equal(t, expectedRecipe.ID, responseRecipe.ID)
	if expectedRecipe.TotalTimeMinutes != nil && responseRecipe.TotalTimeMinutes != nil {
		assert.Equal(t, *expectedRecipe.TotalTimeMinutes, *responseRecipe.TotalTimeMinutes)
	} else {
		assert.Equal(t, expectedRecipe.TotalTimeMinutes, responseRecipe.TotalTimeMinutes)
	}
}

func TestRecipeHandler_UpdateRecipe_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeReq := &models.RecipeRequest{Title: "Valid Title"} // Minimal valid body
	jsonBody, _ := json.Marshal(recipeReq)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/invalid-uuid", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Check error message if needed
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Invalid recipe ID format")
}

func TestRecipeHandler_UpdateRecipe_BindError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)
	recipeID := uuid.New()

	invalidJsonBody := []byte(`{"title": "Test"invalid}`)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/"+recipeID.String(), bytes.NewBuffer(invalidJsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecipeHandler_UpdateRecipe_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)
	recipeID := uuid.New()

	// Request missing required 'Title'
	recipeReq := &models.RecipeRequest{Description: strPtr("Test Description")}
	jsonBody, _ := json.Marshal(recipeReq)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/"+recipeID.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["details"].(map[string]interface{}), "Title")
}

func TestRecipeHandler_UpdateRecipe_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeID := uuid.New()
	recipeReq := &models.RecipeRequest{
		Title: "Valid Title for Not Found Test",
		Serves: intPtr(1), // Ensure it's a valid request
		PrepTimeMinutes: intPtr(1),
		CookTimeMinutes: intPtr(1),
		Ingredients: []models.RecipeIngredientRequest{{IngredientName: "i", Quantity: float64Ptr(1), UnitName: strPtr("u"), SortOrder: 1}},
		Steps: []models.RecipeStepRequest{{StepNumber: 1, Instruction: "s"}},
	}
	jsonBody, _ := json.Marshal(recipeReq)

	// Simulate store returning a "not found" error
	// Note: The actual error string might come from your store.ErrNotFound or similar
	mockStore.EXPECT().UpdateRecipe(gomock.Any(), recipeID, recipeReq).Return(nil, errors.New("recipe not found")).Times(1)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/"+recipeID.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code) // Handler should convert "not found" to 404
}

func TestRecipeHandler_UpdateRecipe_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockRecipeStore(ctrl)
	recipeHandler := NewRecipeHandler(mockStore)
	router := setupTestRouter(recipeHandler)

	recipeID := uuid.New()
	recipeReq := &models.RecipeRequest{
		Title: "Valid Title for Store Error Test",
		Serves: intPtr(1),
		PrepTimeMinutes: intPtr(1),
		CookTimeMinutes: intPtr(1),
		Ingredients: []models.RecipeIngredientRequest{{IngredientName: "i", Quantity: float64Ptr(1), UnitName: strPtr("u"), SortOrder: 1}},
		Steps: []models.RecipeStepRequest{{StepNumber: 1, Instruction: "s"}},
	}
	jsonBody, _ := json.Marshal(recipeReq)

	expectedStoreErrorMessage := "generic database error"
	mockStore.EXPECT().UpdateRecipe(gomock.Any(), recipeID, recipeReq).Return(nil, errors.New(expectedStoreErrorMessage)).Times(1)

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/recipes/"+recipeID.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Failed to update recipe")
	assert.Contains(t, errorResponse["error"], expectedStoreErrorMessage)
}


