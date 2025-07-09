package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	env "github.com/Lifailon/ssh-bot/pkg/env"

	sshClient "golang.org/x/crypto/ssh"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SSH struct {
	PWD             string
	SSH_MODE        bool
	SSH_HOST        string
	SSH_USER        string
	SSH_PORT        string
	SSH_PRIVATE_KEY []byte
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

func (ssh *SSH) sshRunExecCommand(command string, env *env.Env) ([]byte, error) {
	output, err := exec.Command(
		"ssh",
		"-n",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout="+env.SSH_CONNECT_TIMEOUT,
		ssh.SSH_USER+"@"+ssh.SSH_HOST,
		"-p", ssh.SSH_PORT,
		env.LINUX_SHELL, "-c",
		"'"+command+"'",
	).CombinedOutput()
	return output, err
}

// Run command via SSH
func (ssh *SSH) sshRunCommand(command string, env *env.Env) ([]byte, error) {
	// Get signer from private key
	signer, _ := sshClient.ParsePrivateKey(ssh.SSH_PRIVATE_KEY)

	// Convert timeout param
	timeoutSeconds, _ := strconv.Atoi(env.SSH_CONNECT_TIMEOUT)
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second

	// SSH client config
	config := &sshClient.ClientConfig{
		User:            ssh.SSH_USER,
		HostKeyCallback: sshClient.InsecureIgnoreHostKey(), // StrictHostKeyChecking=no
		Timeout:         time.Duration(timeoutDuration),    // ConnectTimeout
		Auth: []sshClient.AuthMethod{
			sshClient.Password(env.SSH_PASSWORD),
		},
	}

	// SSH auth config
	if signer != nil {
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
		log.Println("[WARN] Failed establishing tcp connection")
		return nil, fmt.Errorf("Failed establishing tcp connection: %w", err)
	}
	defer client.Close()

	// Creating SSH session
	session, err := client.NewSession()
	if err != nil {
		log.Println("[WARN] Failed creating ssh session")
		return nil, fmt.Errorf("Failed creating ssh session: %w", err)
	}
	defer session.Close()

	// Run command
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err = session.Run(command)
	if len(stderrBuf.Bytes()) > 0 {
		return stderrBuf.Bytes(), fmt.Errorf("Execution error (error output is not empty)")
	}
	return stdoutBuf.Bytes(), err
}

// Run command on local or remote host
func (ssh *SSH) runCommand(env *env.Env, bot *api.BotAPI, update api.Update, chatID int64, messageText string) {
	var output []byte
	var err error
	var SHELL string
	if ssh.SSH_MODE {
		var command string
		// Import declare from temp file
		if env.SSH_SAVE_ENV {
			command = "[ -e /tmp/ssh-bot.temp ] && source /tmp/ssh-bot.temp; "
		}
		// Change directory + run command
		command = command + "cd " + ssh.PWD + " && " + messageText
		// Export declare in temp file
		if env.SSH_SAVE_ENV {
			command = command + " && declare -p | grep '^declare -- ' > /tmp/ssh-bot.temp; declare -f >> /tmp/ssh-bot.temp"
		}
		output, err = ssh.sshRunCommand(command, env)
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
		out = "`" + ssh.SSH_HOST + "`\n```" + env.LINUX_SHELL + "\n" + string(output) + "```"
	} else {
		// Update shell for Markdown
		if SHELL == "pwsh" {
			SHELL = "powershell"
		}
		out = "`localhost`\n```" + SHELL + "\n" + string(output) + "```"
	}
	if err != nil {
		out = "⚠ " + out
		msg := api.NewMessage(chatID, out)
		msg.ReplyToMessageID = update.Message.MessageID
		msg.ParseMode = api.ModeMarkdown
		bot.Send(msg)
		log.Printf("[ERROR] Execution error on %s: %s", ssh.SSH_HOST, string(output))
	} else {
		out = "▶ " + out
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

	// Read private key
	ssh.SSH_PRIVATE_KEY, _ = os.ReadFile(env.SSH_PRIVATE_KEY_PATH)

	bot, err := api.NewBotAPI(env.TELEGRAM_BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	u := api.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	// Main menu
	commands := []api.BotCommand{
		{Command: "host_list", Description: "List of hosts for ssh connection"},
		{Command: "exit", Description: "Disconnect from the remote host and clear the declared environment"},
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

		// Disconnect from ssh and clear declared environment (remove temp file)
		if messageText == "/exit" || messageText == "exit" {
			if ssh.SSH_MODE {
				if env.SSH_SAVE_ENV {
					env.SSH_SAVE_ENV = false
					ssh.sshRunCommand("rm /tmp/ssh-bot.temp", env)
					env.SSH_SAVE_ENV = true
				}
				ssh.SSH_MODE = false
				messageOutput := api.NewMessage(chatID, "Disconnect from `"+ssh.SSH_HOST+"`")
				messageOutput.ParseMode = api.ModeMarkdown
				bot.Send(messageOutput)
				log.Println("[INFO] Disconnect from" + ssh.SSH_HOST)
			} else {
				bot.Send(api.NewMessage(chatID, "Remote connection not established"))
				log.Println("[INFO] Remote connection not established")
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
			output, err := ssh.sshRunCommand("uname -a", env)
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
				output, _ = ssh.sshRunCommand("pwd", env)
				ssh.PWD = strings.TrimSpace(string(output))
			}
			continue
		}

		// Change directory
		if strings.HasPrefix(messageText, "cd ") {
			// Get path via ssh
			if ssh.SSH_MODE {
				command := "cd " + ssh.PWD + " && " + messageText + " && pwd"
				output, err := ssh.sshRunCommand(command, env)
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
			go ssh.runCommand(env, bot, update, chatID, messageText)
		} else {
			ssh.runCommand(env, bot, update, chatID, messageText)
		}
	}
}
