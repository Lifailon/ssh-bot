package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	env "github.com/Lifailon/ssh-bot/pkg/env"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SSH struct {
	PWD      string
	SSH_MODE bool
	SSH_HOST string
	SSH_USER string
	SSH_PORT string
}

// Get parameters for ssh connection from env
func (ssh *SSH) paramParse(host string, env *env.Env) (string, string, string) {
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

// Execution command on remote host via ssh
func (ssh *SSH) runCommand(command string, env *env.Env) ([]byte, error) {
	output, err := exec.Command(
		"ssh",
		"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
		ssh.SSH_USER+"@"+ssh.SSH_HOST,
		"-p", ssh.SSH_PORT,
		env.LINUX_SHELL, "-c",
		"'"+command+"'",
	).CombinedOutput()
	return output, err
}

// Change directory on localhost
func (ssh *SSH) localChangeDir(bot *api.BotAPI, chatID int64, message string) {
	newPath := strings.TrimSpace(message[3:])
	err := os.Chdir(newPath)
	if err != nil {
		msg := api.NewMessage(chatID, "⚠ Error changing directory:\n\n```Error (go)\n"+err.Error()+"```")
		msg.ParseMode = api.ModeMarkdown
		bot.Send(msg)
		log.Printf("[ERROR] Error changing directory: %s", err.Error())
		return
	}
	pwd, _ := os.Getwd()
	msg := api.NewMessage(chatID, "Current directory:\n\n`"+pwd+"`")
	msg.ParseMode = api.ModeMarkdown
	bot.Send(msg)
	log.Printf("[INFO] Current directory: %s", pwd)
}

func main() {
	log.Println("[INFO] Bot started")

	env := &env.Env{}
	env.GetEnv()

	ssh := &SSH{}

	bot, err := api.NewBotAPI(env.TELEGRAM_BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	u := api.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	// Main menu
	commands := []api.BotCommand{
		{Command: "localhost", Description: "Connect to localhost"},
		{Command: "host_list", Description: "List of hosts for ssh connection"},
	}
	_, err = bot.Request(api.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("[ERROR] %s", string(err.Error()))
	}

	for update := range updates {
		// Get parameters from message/menu and callback query (keyboard)
		var chatID int64
		var firstName string
		var lastName string
		var userName string
		var message string
		switch {
		case update.Message != nil:
			chatID = update.Message.Chat.ID
			firstName = update.Message.Chat.FirstName
			lastName = update.Message.Chat.LastName
			userName = update.Message.Chat.UserName
			message = update.Message.Text
		case update.CallbackQuery != nil:
			chatID = update.CallbackQuery.Message.Chat.ID
			firstName = update.CallbackQuery.From.FirstName
			lastName = update.CallbackQuery.From.LastName
			userName = update.CallbackQuery.From.UserName
			message = update.CallbackQuery.Data
			_, err = bot.Request(api.NewCallback(update.CallbackQuery.ID, ""))
			if err != nil {
				log.Printf("[ERROR] %s", string(err.Error()))
			}
		default:
			continue
		}

		// Access check
		if chatID != env.TELEGRAM_USER_ID {
			bot.Send(api.NewMessage(chatID, "⛔ Access denied ⛔"))
			log.Printf("[WARN] Unauthorized access from %s %s (%s - %d)", firstName, lastName, userName, chatID)
			continue
		}

		log.Printf("[INFO] Request from %s %s (%s - %d): %s", firstName, lastName, userName, chatID, message)

		// Switch to localhost
		if message == "/localhost" {
			ssh.SSH_MODE = false
			messageOutput := api.NewMessage(chatID, "Connection to `localhost`")
			messageOutput.ParseMode = api.ModeMarkdown
			bot.Send(messageOutput)
			log.Println("[INFO] Connection to localhost")
			continue
		}

		// Disconnect from ssh
		if message == "exit" {
			if ssh.SSH_MODE {
				ssh.SSH_MODE = false
				messageOutput := api.NewMessage(chatID, "Disconnect from `"+ssh.SSH_HOST+"`")
				messageOutput.ParseMode = api.ModeMarkdown
				bot.Send(messageOutput)
				log.Println("[INFO] Disconnect from" + ssh.SSH_HOST)
			}
			continue
		}

		// Send keyboard buttons from host list
		if message == "/host_list" {
			var keyboardButton [][]api.InlineKeyboardButton
			for _, host := range env.SSH_HOST_LIST {
				btn := api.NewInlineKeyboardButtonData(host, "/ssh "+host)
				keyboardButton = append(keyboardButton, []api.InlineKeyboardButton{btn})
			}
			keyboard := api.NewInlineKeyboardMarkup(keyboardButton...)
			msg := api.NewMessage(chatID, "Select host to ssh connection:")
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
			continue
		}

		// Switch to selected host via ssh
		if strings.HasPrefix(message, "/ssh") {
			ssh.SSH_MODE = true
			selectedHost := strings.TrimSpace(strings.Replace(message, "/ssh", "", 1))
			if len(selectedHost) == 0 {
				messageOutput := api.NewMessage(chatID, "Host name not specified\n\nPass the host name as a parameter, for example: `/ssh 127.0.0.1`")
				messageOutput.ParseMode = api.ModeMarkdown
				bot.Send(messageOutput)
				log.Println("[ERROR] Host name not specified")
				continue
			}
			ssh.SSH_HOST, ssh.SSH_USER, ssh.SSH_PORT = ssh.paramParse(selectedHost, env)
			sendMessage, _ := bot.Send(api.NewMessage(chatID, "Connection to "+selectedHost))
			lastMessageID := sendMessage.MessageID
			log.Println("[INFO] Connection to " + selectedHost)
			output, err := ssh.runCommand("uname -a", env)
			if err != nil {
				msg := "⚠ Connection error to " + selectedHost + "\n\n" + "```Error\n" + string(output) + "```"
				editMessage := api.NewEditMessageText(chatID, lastMessageID, msg)
				editMessage.ParseMode = api.ModeMarkdown
				bot.Send(editMessage)
				log.Println("[ERROR] Connection error: " + string(output))
			} else {
				msg := "✅ Connection successful to " + selectedHost + "\n\n" + "```Info\n" + string(output) + "```"
				editMessage := api.NewEditMessageText(chatID, lastMessageID, msg)
				editMessage.ParseMode = api.ModeMarkdown
				bot.Send(editMessage)
				log.Println("[INFO] Connection successful")
				output, _ = ssh.runCommand("pwd", env)
				ssh.PWD = strings.TrimSpace(string(output))
			}
			continue
		}

		// Change directory
		if strings.HasPrefix(message, "cd ") {
			// Get path via ssh
			if ssh.SSH_MODE {
				command := "cd " + ssh.PWD + " && " + message + " && pwd"
				output, err := ssh.runCommand(command, env)
				if err != nil {
					msg := api.NewMessage(chatID, "⚠ Error changing directory:\n\n```Error (ssh)\n"+string(output)+"```")
					msg.ParseMode = api.ModeMarkdown
					bot.Send(msg)
					log.Printf("[ERROR] Error changing directory: %s", string(output))
					continue
				}
				ssh.PWD = strings.TrimSpace(string(output))
				msg := api.NewMessage(chatID, "Current directory:\n\n`"+ssh.PWD+"`")
				msg.ParseMode = api.ModeMarkdown
				bot.Send(msg)
				log.Printf("[INFO] Current directory: %s", ssh.PWD)
			} else {
				// Change local directory via os library
				ssh.localChangeDir(bot, chatID, message)
			}
			continue
		}

		// Run command for execution
		var output []byte
		var err error
		var SHELL string
		if ssh.SSH_MODE {
			// Remote (ssh): change directory + run command
			command := "cd " + ssh.PWD + " && " + message
			output, err = ssh.runCommand(command, env)
		} else {
			if runtime.GOOS == "windows" {
				SHELL = env.WIN_SHELL
			} else {
				SHELL = env.LINUX_SHELL
			}
			output, err = exec.Command(SHELL, "-c", message).CombinedOutput()
		}
		var out string
		if ssh.SSH_MODE {
			out = "`" + ssh.SSH_HOST + "`\n\n```" + env.LINUX_SHELL + "\n" + string(output) + "```"
		} else {
			if SHELL == "pwsh" {
				SHELL = "powershell"
			}
			out = "```" + SHELL + "\n" + string(output) + "```"
		}
		if err != nil {
			if ssh.SSH_MODE {
				out = "⚠ Execution error on " + out
			} else {
				out = "⚠ Execution error\n\n" + out
			}
			msg := api.NewMessage(chatID, out)
			msg.ParseMode = api.ModeMarkdown
			bot.Send(msg)
			log.Printf("[ERROR] Execution error %s", out)
			continue
		}
		msg := api.NewMessage(chatID, out)
		msg.ParseMode = api.ModeMarkdown
		bot.Send(msg)
		if env.LOG_MODE == "DEBUG" {
			if ssh.SSH_MODE {
				log.Printf("[DEBUG] Response from %v:", ssh.SSH_HOST)
			} else {
				log.Printf("[DEBUG] Response from localhost:")
			}
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				log.Printf("[DEBUG] %s", line)
			}
		}
	}
}
