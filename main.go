package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or failed to load")
	}
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/recipe", recipeHandler)
	http.HandleFunc("/api/recipe", apiRecipeHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server running on http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func recipeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/recipe.html")
}

func apiRecipeHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	apiKey := os.Getenv("SPOONACULAR_API_KEY")

	if apiKey == "" || id == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	apiURL := fmt.Sprintf("https://api.spoonacular.com/recipes/%s/information?includeNutrition=false&apiKey=%s", id, apiKey)

	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch recipe", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	if query == "" {
		http.Error(w, "Missing query", http.StatusBadRequest)
		return
	}

	page := 1
	pageSize := 10

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	apiKey := os.Getenv("SPOONACULAR_API_KEY")
	spoonURL := fmt.Sprintf("https://api.spoonacular.com/recipes/complexSearch?query=%s&number=100&apiKey=%s", url.QueryEscape(query), apiKey)

	resp, err := http.Get(spoonURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch from Spoonacular", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Image string `json:"image"`
		} `json:"results"`
		TotalResults int `json:"totalResults"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error parsing response", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	total := len(result.Results)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pagedResults := result.Results[start:end]

	// Return paginated results plus total count
	response := map[string]interface{}{
		"results":    pagedResults,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (total + pageSize - 1) / pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
