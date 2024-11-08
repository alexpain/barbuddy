package database

import (
	"database/sql"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
)

type Database struct {
	*sql.DB
}

func New(dataSourceName string) (*Database, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &Database{db}, nil
}

func (db *Database) Stop() {
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}
}

// CreateTable creates the recipes table if it doesn't exist
func (db *Database) CreateTable() error {
	createTablesSQL := `
    CREATE TABLE IF NOT EXISTS recipes (
        recipe_id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        description TEXT
    );

    CREATE TABLE IF NOT EXISTS alcohol_ingredients (
        ingredient_id INTEGER PRIMARY KEY AUTOINCREMENT,
        recipe_id INTEGER,
        name TEXT NOT NULL,
        quantity TEXT,
        FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id)
    );

    CREATE TABLE IF NOT EXISTS non_alcohol_ingredients (
        ingredient_id INTEGER PRIMARY KEY AUTOINCREMENT,
        recipe_id INTEGER,
        name TEXT NOT NULL,
        quantity TEXT,
        FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id)
    );

    CREATE TABLE IF NOT EXISTS garnishes (
        garnish_id INTEGER PRIMARY KEY AUTOINCREMENT,
        recipe_id INTEGER,
        name TEXT NOT NULL,
        FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id)
    );

    CREATE TABLE IF NOT EXISTS utensils (
        utensil_id INTEGER PRIMARY KEY AUTOINCREMENT,
        recipe_id INTEGER,
        name TEXT NOT NULL,
        FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id)
    );

    CREATE TABLE IF NOT EXISTS steps (
        step_id INTEGER PRIMARY KEY AUTOINCREMENT,
        recipe_id INTEGER,
        step_number INTEGER,
        instruction TEXT NOT NULL,
        FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id)
    );`

	_, err := db.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// InsertRecipe inserts a new recipe into the table
func (db *Database) InsertNewRecipe(recipe Recipe) (int64, error) {
	res, err := db.Exec("INSERT INTO recipes (name, description) VALUES (?, ?)", recipe.Name, recipe.Description)
	if err != nil {
		return 0, fmt.Errorf("failed to insert recipe: %v", err)
	}

	recipeID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %v", err)
	}

	for _, ingredient := range recipe.Alcohol {
		_, err = db.Exec("INSERT INTO alcohol_ingredients (recipe_id, name, quantity) VALUES (?, ?, ?)", recipeID, ingredient.Name, ingredient.Quantity)
		if err != nil {
			return 0, fmt.Errorf("failed to insert alcohol ingredient: %v", err)
		}
	}

	for _, ingredient := range recipe.NonAlcohol {
		_, err = db.Exec("INSERT INTO non_alcohol_ingredients (recipe_id, name, quantity) VALUES (?, ?, ?)", recipeID, ingredient.Name, ingredient.Quantity)
		if err != nil {
			return 0, fmt.Errorf("failed to insert non-alcohol ingredient: %v", err)
		}
	}

	for _, garnish := range recipe.Garnishes {
		_, err = db.Exec("INSERT INTO garnishes (recipe_id, name) VALUES (?, ?)", recipeID, garnish)
		if err != nil {
			return 0, fmt.Errorf("failed to insert garnish: %v", err)
		}
	}

	for _, utensil := range recipe.Utensils {
		_, err = db.Exec("INSERT INTO utensils (recipe_id, name) VALUES (?, ?)", recipeID, utensil)
		if err != nil {
			return 0, fmt.Errorf("failed to insert utensil: %v", err)
		}
	}

	for stepNumber, step := range recipe.Steps {
		_, err = db.Exec("INSERT INTO steps (recipe_id, step_number, instruction) VALUES (?, ?, ?)", recipeID, stepNumber+1, step)
		if err != nil {
			return 0, fmt.Errorf("failed to insert step %d: %v", stepNumber+1, err)
		}
	}

	return recipeID, nil
}

// GetAllRecipes retrieves all recipes from the table
func (db *Database) GetAllRecipes() ([]RecipeWithDetails, error) {
	rows, err := db.Query("SELECT recipe_id, name, description FROM recipes")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipes: %v", err)
	}
	defer rows.Close()

	var recipes []RecipeWithDetails

	for rows.Next() {
		var recipe RecipeWithDetails
		if err := rows.Scan(&recipe.ID, &recipe.Name, &recipe.Description); err != nil {
			return nil, fmt.Errorf("failed to scan recipe: %v", err)
		}

		recipe.Alcohol, err = db.getIngredients(recipe.ID, "alcohol_ingredients")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch alcohol ingredients: %v", err)
		}

		recipe.NonAlcohol, err = db.getIngredients(recipe.ID, "non_alcohol_ingredients")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch non-alcohol ingredients: %v", err)
		}

		recipe.Garnishes, err = db.getGarnishes(recipe.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch garnishes: %v", err)
		}

		recipe.Utensils, err = db.getUtensils(recipe.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch utensils: %v", err)
		}

		recipe.Steps, err = db.getSteps(recipe.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch steps: %v", err)
		}

		recipes = append(recipes, recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating over rows: %v", err)
	}

	return recipes, nil
}

// Вспомогательная функция для получения ингредиентов
func (db *Database) getIngredients(recipeID int64, table string) ([]Ingredient, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT name, quantity FROM %s WHERE recipe_id = ?", table), recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingredients: %v", err)
	}
	defer rows.Close()

	var ingredients []Ingredient
	for rows.Next() {
		var ingredient Ingredient
		if err := rows.Scan(&ingredient.Name, &ingredient.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan ingredient: %v", err)
		}
		ingredients = append(ingredients, ingredient)
	}
	return ingredients, nil
}

func (db *Database) getGarnishes(recipeID int64) ([]string, error) {
	rows, err := db.Query("SELECT name FROM garnishes WHERE recipe_id = ?", recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch garnishes: %v", err)
	}
	defer rows.Close()

	var garnishes []string
	for rows.Next() {
		var garnish string
		if err := rows.Scan(&garnish); err != nil {
			return nil, fmt.Errorf("failed to scan garnish: %v", err)
		}
		garnishes = append(garnishes, garnish)
	}
	return garnishes, nil
}

// Вспомогательная функция для получения посуды
func (db *Database) getUtensils(recipeID int64) ([]string, error) {
	rows, err := db.Query("SELECT name FROM utensils WHERE recipe_id = ?", recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch utensils: %v", err)
	}
	defer rows.Close()

	var utensils []string
	for rows.Next() {
		var utensil string
		if err := rows.Scan(&utensil); err != nil {
			return nil, fmt.Errorf("failed to scan utensil: %v", err)
		}
		utensils = append(utensils, utensil)
	}
	return utensils, nil
}

// Вспомогательная функция для получения шагов
func (db *Database) getSteps(recipeID int64) ([]string, error) {
	rows, err := db.Query("SELECT instruction FROM steps WHERE recipe_id = ? ORDER BY step_number", recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch steps: %v", err)
	}
	defer rows.Close()

	var steps []string
	for rows.Next() {
		var step string
		if err := rows.Scan(&step); err != nil {
			return nil, fmt.Errorf("failed to scan step: %v", err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}
