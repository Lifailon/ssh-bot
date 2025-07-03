package fs

import (
	"log"
	"os"
	"strings"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Fs struct {
	pwd string
}

func (fs *Fs) ChangeDir(bot *api.BotAPI, chatID int64, message string) {
	newPath := strings.TrimSpace(message[3:])
	err := os.Chdir(newPath)
	if err != nil {
		bot.Send(api.NewMessage(chatID, "Execution error:\n\n"+err.Error()))
		log.Printf("[ERROR] Error changing directory: %s", err.Error())
	}
	pwd, _ := os.Getwd()
	fs.pwd = pwd
	bot.Send(api.NewMessage(chatID, "Current directory:\n\n"+pwd))
	log.Printf("[INFO] Current directory: %s", pwd)
}
