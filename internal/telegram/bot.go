package telegram

import (
	"fmt"
	"github.com/alexpain/barbuddy/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

var userStates = make(map[int64]*UserState)

type UserState struct {
	Step        int
	Recipe      database.Recipe
	Current     string
	Ingredients []database.Ingredient
}

type Bot struct {
	*tgbotapi.BotAPI
	db *database.Database
}

func NewBot(token string, db *database.Database) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
		return nil, err
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		BotAPI: bot,
		db:     db,
	}, nil
}

func (b *Bot) Update() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome!")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			case "help":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "List of available commands: /start, /help")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			case "get_recipes":
				recipes, err := b.db.GetAllRecipes()
				if err != nil {
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error getting recipes: %v", err)))
					continue
				}
				if len(recipes) == 0 {
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "No recipes found."))
					continue
				}
				var response strings.Builder
				for _, recipe := range recipes {
					response.WriteString(fmt.Sprintf("Recipe: %s\nDescription: %s\n", recipe.Name, recipe.Description))
					response.WriteString("Alcohol Ingredients:\n")
					for _, ingredient := range recipe.Alcohol {
						response.WriteString(fmt.Sprintf(" - %s: %s\n", ingredient.Name, ingredient.Quantity))
					}
					response.WriteString("Non-Alcohol Ingredients:\n")
					for _, ingredient := range recipe.NonAlcohol {
						response.WriteString(fmt.Sprintf(" - %s: %s\n", ingredient.Name, ingredient.Quantity))
					}
					response.WriteString("Garnishes:\n")
					for _, garnish := range recipe.Garnishes {
						response.WriteString(fmt.Sprintf(" - %s\n", garnish))
					}
					response.WriteString("Utensils:\n")
					for _, utensil := range recipe.Utensils {
						response.WriteString(fmt.Sprintf(" - %s\n", utensil))
					}
					response.WriteString("Steps:\n")
					for i, step := range recipe.Steps {
						response.WriteString(fmt.Sprintf(" %d. %s\n", i+1, step))
					}
					response.WriteString("------------------------------------------------------\n")
				}
				b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, response.String()))
			case "add_recipe":
				userStates[update.Message.Chat.ID] = &UserState{
					Step:   1,
					Recipe: database.Recipe{},
				}
				b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Let's start creating a new recipe! What's the name of the recipe?"))
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			}
		} else {
			if state, exists := userStates[update.Message.Chat.ID]; exists {
				switch state.Step {
				case 1:
					state.Recipe.Name = update.Message.Text
					state.Step++
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Great! Now, can you provide a description of the recipe?"))
				case 2:
					state.Recipe.Description = update.Message.Text
					state.Step++
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Next, please provide the alcohol ingredients (e.g., 'Rum: 50ml'). Type 'done' when finished."))
				case 3:
					if update.Message.Text == "done" {
						state.Step++
						b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Great! Now, please provide non-alcohol ingredients (e.g., 'Lime juice: 25ml'). Type 'done' when finished."))
					} else {
						parts := strings.Split(update.Message.Text, ":")
						if len(parts) == 2 {
							state.Ingredients = append(state.Ingredients, database.Ingredient{
								Name:     strings.TrimSpace(parts[0]),
								Quantity: strings.TrimSpace(parts[1]),
							})
						}
					}
				case 4:
					if update.Message.Text == "done" {
						state.Step++
						b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Please provide garnishes (e.g., 'Mint leaves'). Type 'done' when finished."))
					} else {
						state.Recipe.NonAlcohol = append(state.Recipe.NonAlcohol, state.Ingredients...)
						state.Ingredients = nil
						state.Step++
					}
				case 5:
					if update.Message.Text == "done" {
						state.Step++
						b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "What utensils are needed for this recipe? (e.g., 'Shaker, Glass'). Type 'done' when finished."))
					} else {
						state.Recipe.Garnishes = append(state.Recipe.Garnishes, update.Message.Text)
					}
				case 6:
					if update.Message.Text == "done" {
						state.Step++
						b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Please provide the steps for the recipe. Type 'done' when finished."))
					} else {
						state.Recipe.Utensils = append(state.Recipe.Utensils, update.Message.Text)
					}
				case 7:
					if update.Message.Text == "done" {
						state.Step++
						b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "You're almost done! Now, provide the steps for preparing the recipe (e.g., 'Step 1: Muddle mint leaves'). Type 'done' when finished."))
					} else {
						state.Recipe.Steps = append(state.Recipe.Steps, update.Message.Text)
					}
				case 8:
					if update.Message.Text == "done" {
						recipeID, err := b.db.InsertNewRecipe(state.Recipe)
						if err != nil {
							b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error inserting recipe: %v", err)))
						} else {
							b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Your new recipe has been added successfully with ID: %d", recipeID)))
						}
						delete(userStates, update.Message.Chat.ID)
					}
				}
			}
		}
	}

	return nil
}
