package env

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Env struct {
	TELEGRAM_BOT_TOKEN  string
	TELEGRAM_USER_ID    int64
	WIN_SHELL           string
	LINUX_SHELL         string
	SSH_USER            string
	SSH_PORT            string
	SSH_HOSTS           string
	SSH_HOST_LIST       []string
	SSH_CONNECT_TIMEOUT string
	LOG_MODE            string
}

func (env *Env) GetEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {

	}
	dataString := strings.TrimSpace(string(data))
	lines := strings.Split(dataString, "\n")

	var linesNotComments []string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		} else if strings.Contains(line, "=") {
			linesNotComments = append(linesNotComments, line)
		}
	}

	for _, line := range linesNotComments {
		envArr := strings.Split(line, "=")
		envKey := strings.TrimSpace(envArr[0])
		envValue := strings.TrimSpace(envArr[1])
		switch {
		case envKey == "TELEGRAM_BOT_TOKEN":
			env.TELEGRAM_BOT_TOKEN = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "TELEGRAM_USER_ID":
			TELEGRAM_USER_ID_STR := strings.TrimSpace(strings.Split(envValue, "#")[0])
			TELEGRAM_USER_ID_INT, _ := strconv.ParseInt(TELEGRAM_USER_ID_STR, 10, 64)
			env.TELEGRAM_USER_ID = TELEGRAM_USER_ID_INT
		case envKey == "WIN_SHELL":
			env.WIN_SHELL = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "LINUX_SHELL":
			env.LINUX_SHELL = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "SSH_USER":
			env.SSH_USER = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "SSH_PORT":
			env.SSH_PORT = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "SSH_HOST_LIST":
			env.SSH_HOSTS = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "SSH_CONNECT_TIMEOUT":
			env.SSH_CONNECT_TIMEOUT = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "LOG_MODE":
			env.LOG_MODE = strings.TrimSpace(strings.Split(envValue, "#")[0])
		}
	}

	env.GetHost()

	if env.LOG_MODE == "DEBUG" {
		env.PrintEnv()
	}
}

func (env *Env) GetHost() {
	env.SSH_HOST_LIST = []string{}
	hosts := strings.Split(env.SSH_HOSTS, ",")
	for _, host := range hosts {
		env.SSH_HOST_LIST = append(env.SSH_HOST_LIST, strings.TrimSpace(host))
	}
}

func (env *Env) PrintEnv() {
	log.Println()
	log.Println("[ENV] TELEGRAM_BOT_TOKEN: " + env.TELEGRAM_BOT_TOKEN)
	log.Printf("[ENV] TELEGRAM_USER_ID: %d \n", env.TELEGRAM_USER_ID)
	log.Println("[ENV] WIN_SHELL: " + env.WIN_SHELL)
	log.Println("[ENV] LINUX_SHELL: " + env.LINUX_SHELL)
	log.Println("[ENV] SSH_USER: " + env.SSH_USER)
	log.Println("[ENV] SSH_PORT: " + env.SSH_PORT)
	log.Println("[ENV] SSH_HOST_LIST:")
	for _, host := range env.SSH_HOST_LIST {
		log.Println("[ENV] - " + host)
	}
	log.Println("[ENV] SSH_CONNECT_TIMEOUT: " + env.SSH_CONNECT_TIMEOUT)
	log.Println()
}
