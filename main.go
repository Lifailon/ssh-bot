package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	env "github.com/Lifailon/ssh-bot/pkg/env"

	sshClient "golang.org/x/crypto/ssh"

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

// Change directory on localhost
func (ssh *SSH) localChangeDir(bot *api.BotAPI, chatID int64, message string) {
	newPath := strings.TrimSpace(message[3:])
	err := os.Chdir(newPath)
	if err != nil {
		msg := api.NewMessage(chatID, "⚠ Error changing directory:\n\n```Go\n"+err.Error()+"```")
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

// func (ssh *SSH) runCommand(command string, env *env.Env) ([]byte, error) {
// 	output, err := exec.Command(
// 		"ssh",
// 		"-n",
// 		"-o", "StrictHostKeyChecking=no",
// 		"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
// 		ssh.SSH_USER+"@"+ssh.SSH_HOST,
// 		"-p", ssh.SSH_PORT,
// 		env.LINUX_SHELL, "-c",
// 		"'"+command+"'",
// 	).CombinedOutput()
// 	return output, err
// }

func (ssh *SSH) runCommand(command string, env *env.Env) ([]byte, error) {
	// Get Private Key
	var envPath string
	if runtime.GOOS == "windows" {
		envPath = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	} else {
		envPath = os.Getenv("HOME")
	}
	keyFiles, _ := filepath.Glob(envPath + "/.ssh/id_*")
	var signer sshClient.Signer = nil
	for _, keyFile := range keyFiles {
		if strings.HasSuffix(keyFile, ".pub") {
			continue
		}
		data, _ := os.ReadFile(keyFile)
		signer, _ = sshClient.ParsePrivateKey(data)
		break
	}

	// Convert timeout param
	timeoutSeconds, _ := strconv.Atoi(env.SSH_CONNECT_TIMEOUT)
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second

	// SSH main params
	config := &sshClient.ClientConfig{
		User:            ssh.SSH_USER,
		HostKeyCallback: sshClient.InsecureIgnoreHostKey(), // StrictHostKeyChecking=no
		Timeout:         time.Duration(timeoutDuration),    // ConnectTimeout
		Auth: []sshClient.AuthMethod{
			sshClient.Password(env.SSH_PASSWORD),
		},
	}

	// SSH auth params
	if keyFiles != nil {
		config.Auth = []sshClient.AuthMethod{
			sshClient.PublicKeys(signer),
			sshClient.Password(env.SSH_PASSWORD),
		}
	} else {
		log.Println("[WARN] Private key not found")
		if len(env.SSH_PASSWORD) == 0 {
			log.Println("[WARN] Password not set")
		}
	}

	// Establishing TCP connection
	client, err := sshClient.Dial("tcp", fmt.Sprintf("%s:%s", ssh.SSH_HOST, ssh.SSH_PORT), config)
	if err != nil {
		return nil, fmt.Errorf("Failed establishing tcp connection: %w", err)
	}
	defer client.Close()

	// Creating SSH session
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed creating ssh session: %w", err)
	}
	defer session.Close()

	// Run command
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err = session.Run(command)
	if err != nil {
		return append(stderrBuf.Bytes(), stdoutBuf.Bytes()...), err
	}

	return stdoutBuf.Bytes(), nil
}

func (ssh *SSH) runExec(env *env.Env, bot *api.BotAPI, update api.Update, chatID int64, messageText string) {
	var output []byte
	var err error
	var SHELL string
	if ssh.SSH_MODE {
		// Remote (ssh): change directory + run command
		command := "cd " + ssh.PWD + " && " + messageText
		output, err = ssh.runCommand(command, env)
	} else {
		if runtime.GOOS == "windows" {
			SHELL = env.WIN_SHELL
		} else {
			SHELL = env.LINUX_SHELL
		}
		output, err = exec.Command(SHELL, "-c", messageText).CombinedOutput()
	}
	var out string
	if ssh.SSH_MODE {
		out = "Response from `" + ssh.SSH_HOST + "`\n```" + env.LINUX_SHELL + "\n" + string(output) + "```"
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
			out = "⚠ Execution error\n" + out
		}
		msg := api.NewMessage(chatID, out)
		msg.ReplyToMessageID = update.Message.MessageID
		msg.ParseMode = api.ModeMarkdown
		bot.Send(msg)
		log.Printf("[ERROR] Execution error on %s: %s", ssh.SSH_HOST, string(output))
	} else {
		msg := api.NewMessage(chatID, out)
		msg.ReplyToMessageID = update.Message.MessageID
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
		{Command: "localhost", Description: "Connect to localhost (disconnect from remote host)"},
		{Command: "host_list", Description: "List of hosts for ssh connection"},
		{Command: "ssh", Description: "Connect to the specified host (example: /ssh 192.168.1.1)"},
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
		var messageText string
		switch {
		case update.Message != nil:
			chatID = update.Message.Chat.ID
			firstName = update.Message.Chat.FirstName
			lastName = update.Message.Chat.LastName
			userName = update.Message.Chat.UserName
			messageText = update.Message.Text
		case update.CallbackQuery != nil:
			chatID = update.CallbackQuery.Message.Chat.ID
			firstName = update.CallbackQuery.From.FirstName
			lastName = update.CallbackQuery.From.LastName
			userName = update.CallbackQuery.From.UserName
			messageText = update.CallbackQuery.Data
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

		log.Printf("[INFO] Request from %s %s (%s - %d): %s", firstName, lastName, userName, chatID, messageText)

		// Switch to localhost
		if messageText == "/localhost" {
			ssh.SSH_MODE = false
			messageOutput := api.NewMessage(chatID, "Connection to `localhost`")
			messageOutput.ParseMode = api.ModeMarkdown
			bot.Send(messageOutput)
			log.Println("[INFO] Connection to localhost")
			continue
		}

		// Disconnect from ssh
		if messageText == "exit" {
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
		if messageText == "/host_list" {
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
		if strings.HasPrefix(messageText, "/ssh") {
			ssh.SSH_MODE = true
			selectedHost := strings.TrimSpace(strings.Replace(messageText, "/ssh", "", 1))
			if len(selectedHost) == 0 {
				messageOutput := api.NewMessage(chatID, "Host name not specified\n\nPass the host name as a parameter, for example: `/ssh 192.168.1.1`")
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
		if strings.HasPrefix(messageText, "cd ") {
			// Get path via ssh
			if ssh.SSH_MODE {
				command := "cd " + ssh.PWD + " && " + messageText + " && pwd"
				output, err := ssh.runCommand(command, env)
				if err != nil {
					msg := api.NewMessage(chatID, "⚠ Error changing directory:\n\n```ssh\n"+string(output)+"```")
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
				ssh.localChangeDir(bot, chatID, messageText)
			}
			continue
		}

		// Run command for execution
		if env.PARALLEL_EXEC {
			go ssh.runExec(env, bot, update, chatID, messageText)
		} else {
			ssh.runExec(env, bot, update, chatID, messageText)
		}
	}
}
