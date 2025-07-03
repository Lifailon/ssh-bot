package env

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Env struct {
	TELEGRAM_BOT_TOKEN string
	TELEGRAM_USER_ID   int64
	LOG_LEVEL          string
	WIN_SHELL          string
	LINUX_SHELL        string
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
		case envKey == "LOG_LEVEL":
			env.LOG_LEVEL = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "WIN_SHELL":
			env.WIN_SHELL = strings.TrimSpace(strings.Split(envValue, "#")[0])
		case envKey == "LINUX_SHELL":
			env.LINUX_SHELL = strings.TrimSpace(strings.Split(envValue, "#")[0])
		}
	}

	if env.LOG_LEVEL == "DEBUG" {
		log.Println()
		log.Println("TELEGRAM_BOT_TOKEN: " + env.TELEGRAM_BOT_TOKEN)
		log.Printf("TELEGRAM_USER_ID: %d \n", env.TELEGRAM_USER_ID)
		log.Println("WIN_SHELL: " + env.WIN_SHELL)
		log.Println("LINUX_SHELL: " + env.LINUX_SHELL)
		log.Println()
	}
}
