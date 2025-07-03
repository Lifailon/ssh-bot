package main

import (
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Lifailon/ssh-bot/pkg/env"
	"github.com/Lifailon/ssh-bot/pkg/fs"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	log.Println("[INFO] Bot started")

	env := &env.Env{}
	env.GetEnv()

	fs := &fs.Fs{}

	bot, err := api.NewBotAPI(env.TELEGRAM_BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	u := api.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		firstName := update.Message.Chat.FirstName
		LastName := update.Message.Chat.LastName
		userName := update.Message.Chat.UserName
		message := update.Message.Text

		if chatID != env.TELEGRAM_USER_ID {
			bot.Send(api.NewMessage(chatID, "Access denied"))
			log.Printf("[WARN] Unauthorized access from %s %s (%s - %d)", firstName, LastName, userName, chatID)
			continue
		}

		log.Printf("[INFO] Executing command from %s %s (%s - %d): %s", firstName, LastName, userName, chatID, message)

		if strings.HasPrefix(message, "cd ") {
			fs.ChangeDir(bot, chatID, message)
			continue
		}

		var output []byte
		var err error
		if runtime.GOOS == "windows" {
			output, err = exec.Command(env.WIN_SHELL, "-command", message).CombinedOutput()
		} else {
			output, err = exec.Command(env.LINUX_SHELL, "-c", message).CombinedOutput()
		}

		if err != nil {
			// bot.Send(api.NewMessage(chatID, "Execution error: "+err.Error()))
			bot.Send(api.NewMessage(chatID, "Execution error:\n\n"+string(output)))
			log.Printf("[ERROR] Execution error: %s", string(output))
			continue
		}

		bot.Send(api.NewMessage(chatID, string(output)))
		if env.LOG_LEVEL == "DEBUG" {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				log.Printf("[DEBUG] %s", line)
			}
		}
	}
}
