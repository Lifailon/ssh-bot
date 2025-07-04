package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Lifailon/ssh-bot/pkg/env"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Param struct {
	PWD      string
	SSH_MODE bool
	SSH_HOST string
	SSH_USER string
	SSH_PORT string
}

func (param *Param) sshParamParse(host string, env *env.Env) (string, string, string) {
	var userName, port string
	if strings.Contains(host, "@") {
		hostSplit := strings.Split(host, "@")
		userName = hostSplit[0]
		host = hostSplit[1]
	} else {
		userName = env.SSH_USER
	}
	if strings.Contains(host, ":") {
		hostSplit := strings.Split(host, ":")
		port = hostSplit[1]
		host = hostSplit[0]
	} else {
		port = env.SSH_PORT
	}
	return host, userName, port
}

func (param *Param) changeDir(bot *api.BotAPI, chatID int64, message string) {
	newPath := strings.TrimSpace(message[3:])
	err := os.Chdir(newPath)
	if err != nil {
		bot.Send(api.NewMessage(chatID, "Execution error:\n\n"+err.Error()))
		log.Printf("[ERROR] Error changing directory: %s", err.Error())
		return
	}
	pwd, _ := os.Getwd()
	param.PWD = pwd
	bot.Send(api.NewMessage(chatID, "Current directory:\n\n"+pwd))
	log.Printf("[INFO] Current directory: %s", pwd)
}

func main() {
	log.Println("[INFO] Bot started")

	env := &env.Env{}
	env.GetEnv()

	param := &Param{}

	bot, err := api.NewBotAPI(env.TELEGRAM_BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	u := api.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	commands := []api.BotCommand{
		{Command: "localhost", Description: "Connect to localhost"},
		{Command: "host_list", Description: "List of hosts for ssh connection"},
	}
	_, err = bot.Request(api.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("[ERROR] %s", string(err.Error()))
	}

	for update := range updates {
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

		log.Printf("[INFO] Request from %s %s (%s - %d): %s", firstName, LastName, userName, chatID, message)

		if message == "/localhost" {
			param.SSH_MODE = false
			bot.Send(api.NewMessage(chatID, "Connection to localhost"))
			log.Println("[INFO] Connection to localhost")
			continue
		}

		if message == "/host_list" {
			response := "List of hosts for ssh connection\n\n"
			for _, host := range env.SSH_HOST_LIST {
				response += "`/ssh " + host + "`\n"
			}
			msg := api.NewMessage(chatID, response)
			msg.ParseMode = api.ModeMarkdown
			bot.Send(msg)
			continue
		}

		if strings.HasPrefix(message, "/ssh") {
			param.SSH_MODE = true
			selectedHost := strings.TrimSpace(strings.Replace(message, "/ssh", "", 1))
			param.SSH_HOST, param.SSH_USER, param.SSH_PORT = param.sshParamParse(selectedHost, env)
			bot.Send(api.NewMessage(chatID, "Connection to "+selectedHost))
			log.Println("[INFO] Connection to " + selectedHost)
			output, err := exec.Command(
				"ssh",
				"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
				param.SSH_USER+"@"+param.SSH_HOST,
				"-p", param.SSH_PORT,
				env.LINUX_SHELL, "-c",
				"'"+"uname -a"+"'",
			).CombinedOutput()
			if err != nil {
				// Backchange last message
				log.Println("Connection error: " + string(output))
				bot.Send(api.NewMessage(chatID, "Connection error:\n\n"+string(output)))
			} else {
				log.Println("Connection successful")
				bot.Send(api.NewMessage(chatID, "Connection successful:\n\n"+string(output)))
				output, err = exec.Command(
					"ssh",
					"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
					param.SSH_USER+"@"+param.SSH_HOST,
					"-p", param.SSH_PORT,
					env.LINUX_SHELL, "-c",
					"'"+"pwd"+"'",
				).CombinedOutput()
				param.PWD = string(output)
			}
			continue
		}

		if strings.HasPrefix(message, "cd ") {
			if param.SSH_MODE {

			} else {
				param.changeDir(bot, chatID, message)
			}
			continue
		}

		var output []byte
		var err error
		var SHELL string
		if param.SSH_MODE {
			// prep cd
			output, err = exec.Command(
				"ssh",
				"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
				param.SSH_USER+"@"+param.SSH_HOST,
				"-p", param.SSH_PORT,
				env.LINUX_SHELL, "-c",
				"'"+message+"'",
			).CombinedOutput()
		} else {
			if runtime.GOOS == "windows" {
				SHELL = env.WIN_SHELL
			} else {
				SHELL = env.LINUX_SHELL
			}
			output, err = exec.Command(SHELL, "-c", message).CombinedOutput()
		}

		if err != nil {
			bot.Send(api.NewMessage(chatID, "Execution error:\n\n"+string(output)))
			log.Printf("[ERROR] Execution error: %s", string(output))
			continue
		}

		bot.Send(api.NewMessage(chatID, string(output)))
		if env.LOG_MODE == "DEBUG" {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				log.Printf("[DEBUG] %s", line)
			}
		}
	}
}
