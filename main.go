package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	token       = "XXXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	userID      = 7777777777
	logLevel    = "DEBUG"
	WIN_SHELL   = "pwsh" // pwsh/powershell
	LINUX_SHELL = "bash" // bash/sh and other
)

func main() {
	bot, err := api.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := api.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	log.Println("[INFO] Bot started")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		firstName := update.Message.Chat.FirstName
		LastName := update.Message.Chat.LastName
		userName := update.Message.Chat.UserName
		message := update.Message.Text

		if chatID != userID {
			bot.Send(api.NewMessage(chatID, "Доступ запрещен"))
			log.Printf("[WARN] Unauthorized access from %s %s (%s - %d)", firstName, LastName, userName, chatID)
			continue
		}

		log.Printf("[INFO] Executing command from %s %s (%s - %d): %s", firstName, LastName, userName, chatID, message)

		if strings.HasPrefix(message, "cd ") {
			newPath := strings.TrimSpace(message[3:])
			err := os.Chdir(newPath)
			if err != nil {
				bot.Send(api.NewMessage(chatID, "Ошибка выполнения:\n\n"+err.Error()))
				log.Printf("[ERROR] Error changing directory: %v", err.Error())
				continue
			}
			pwd, _ := os.Getwd()
			bot.Send(api.NewMessage(chatID, "Текущая директория:\n\n"+pwd))
			log.Printf("[INFO] Current directory: %v", pwd)
			continue
		}

		var output []byte
		var err error
		if runtime.GOOS == "windows" {
			output, err = exec.Command(WIN_SHELL, "-Command", message).CombinedOutput()
		} else {
			// Linux Bash/Shell
			output, err = exec.Command(LINUX_SHELL, "-c", message).CombinedOutput()
		}

		if err != nil {
			// bot.Send(api.NewMessage(chatID, "Ошибка выполнения: "+err.Error()))
			bot.Send(api.NewMessage(chatID, "Ошибка выполнения:\n\n"+string(output)))
			log.Printf("[ERROR] Execution error: %v", string(output))
			continue
		}

		bot.Send(api.NewMessage(chatID, string(output)))
		if logLevel == "DEBUG" {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				log.Printf("[DEBUG] %v", line)
			}
		}
	}

	log.Println("[INFO] Bot stopped")
}
