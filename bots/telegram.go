package bots

import (
	"fmt"
	"github.com/alexpain/barbuddy/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

var userStates = make(map[int64]*UserState)

type repositoryType interface {
	InsertNewRecipe(recipe database.Recipe) (int64, error)
	GetAllRecipes() ([]database.RecipeWithDetails, error)
}

type UserState struct {
	Step    int
	Recipe  database.Recipe
	Current string
}

type Bot struct {
	*tgbotapi.BotAPI
	repository repositoryType
}

func NewTelegramBot(token string, repository repositoryType) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
		return nil, err
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		BotAPI:     bot,
		repository: repository,
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
				delete(userStates, update.Message.Chat.ID)
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
				delete(userStates, update.Message.Chat.ID)
				recipes, err := b.repository.GetAllRecipes()
				if err != nil {
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error getting recipes: %v", err)))
					continue
				}
				if len(recipes) == 0 {
					b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "No recipes found."))
					continue
				}
				response := b.formatRecipes(recipes)
				b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, response.String()))
			case "add_recipe":
				delete(userStates, update.Message.Chat.ID)
				userStates[update.Message.Chat.ID] = &UserState{
					Step:   1,
					Recipe: database.Recipe{},
				}
				b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Let's start creating a new recipe! What's the name of the recipe?"))
			case "cancel":
				delete(userStates, update.Message.Chat.ID)
				b.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Current operation cancelled"))
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			}
		} else {
			b.processSteps(update)
		}
	}

	return nil
}

func (b *Bot) processSteps(update tgbotapi.Update) {
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
					state.Recipe.Alcohol = append(state.Recipe.Alcohol, database.Ingredient{
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
				parts := strings.Split(update.Message.Text, ":")
				if len(parts) == 2 {
					state.Recipe.NonAlcohol = append(state.Recipe.NonAlcohol, database.Ingredient{
						Name:     strings.TrimSpace(parts[0]),
						Quantity: strings.TrimSpace(parts[1]),
					})
				}
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
				state.Recipe.UserId = update.Message.Chat.ID
				recipeID, err := b.repository.InsertNewRecipe(state.Recipe)
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

func (b *Bot) formatRecipes(recipes []database.RecipeWithDetails) strings.Builder {
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
	return response
}
