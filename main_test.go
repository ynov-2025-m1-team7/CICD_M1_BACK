package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupApp() *fiber.App {
	app := fiber.New()

	// Define routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API de gestion des feedbacks")
	})

	app.Get("/feedbacks", func(c *fiber.Ctx) error {
		return c.JSON([]map[string]interface{}{})
	})

	app.Post("/feedbacks", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{"inserted_id": "12345"})
	})

	app.Put("/feedbacks/:id", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{"modified_count": 1})
	})

	app.Delete("/feedbacks/:id", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{"deleted_count": 1})
	})

	app.Get("/feedbacks/average-score", func(c *fiber.Ctx) error {
		return c.JSON(map[string]float64{"average_score": 4.5})
	})

	app.Get("/generate-error", func(c *fiber.Ctx) error {
		return c.Status(500).JSON(map[string]string{"error": "This is a test error"})
	})

	return app
}

func TestRootEndpoint(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestListFeedbacks(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/feedbacks", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestAddFeedback(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodPost, "/feedbacks", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestUpdateFeedback(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodPut, "/feedbacks/12345", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDeleteFeedback(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodDelete, "/feedbacks/12345", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestAverageScore(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/feedbacks/average-score", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestGenerateError(t *testing.T) {
	app := setupApp()
	req := httptest.NewRequest(http.MethodGet, "/generate-error", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
}

// Custom test runner to output results in JSON format
func TestMain(m *testing.M) {
	// Run the tests
	result := m.Run()

	// Create a JSON file to store the results
	file, err := os.Create("test-results.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write test results to the JSON file
	json.NewEncoder(file).Encode(map[string]interface{}{
		"result": result,
	})

	os.Exit(result)
}
