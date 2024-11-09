package database

type Recipe struct {
	UserId      int64
	Name        string
	Description string
	Alcohol     []Ingredient
	NonAlcohol  []Ingredient
	Garnishes   []string
	Utensils    []string
	Steps       []string
}

type Ingredient struct {
	Name     string
	Quantity string
}

type RecipeWithDetails struct {
	ID          int64
	Name        string
	Description string
	Alcohol     []Ingredient
	NonAlcohol  []Ingredient
	Garnishes   []string
	Utensils    []string
	Steps       []string
}
