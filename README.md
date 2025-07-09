<h1 align="center">
    <img src="img/logo.png" width="220" />
    <div>
    SSH Bot
    </div>
</h1>

<h4 align="center">
    <strong>English (🇺🇸)</strong> | <a href="README_RU.md">Русский (🇷🇺)</a>
</h4>

Telegram bot that allows you to run specified commands on a selected host in your home network and return the result of their execution. The bot does not establish a permanent connection with the remote host, which allows you to execute commands asynchronously.

The bot provides the ability to not waste time setting up a `VPN` server and money on an external IP address or `VPS` server to access the local network, and also eliminates the need to use third-party applications (`VPN` and `ssh` clients) on a remote device and does not require a stable Internet connection.

![example](/img/demo.gif)

## Roadmap

- [X] Executing commands on the local (the one where the bot is running) or remote host (via `ssh`) in the specified interpreter.
- [X] Support for parallel (asynchronous) command execution.
- [X] `ssh` connection manager with host availability check.
- [X] Support for directory navigation.
- [X] Combined access to remote hosts by key and/or password.
- [X] Error handling when using commands that require user input.
- [X] Support for storing and reusing passed variables and functions (the `/exit` command clears the history).

## Launch

You can download the pre-compiled executable from the [releases](https://github.com/Lifailon/ssh-bot/releases) page and run the bot locally or in a Docker container using the image from [Docker Hub](https://hub.docker.com/r/lifailon/ssh-bot).

> [!NOTE]
> Before launching, you need to create your Telegram bot using [@BotFather](https://telegram.me/BotFather) and get its `API Token`, which must be specified in the configuration file.

- Create a working directory:

```shell
mkdir ssh-bot
cd ssh-bot
```

- Create and fill the `.env` file file inside the working directory:

```shell
TELEGRAM_BOT_TOKEN=XXXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
TELEGRAM_USER_ID=7777777777

# Interpreter used only when running the bot local in Windows
# Available values: powershell/pwsh
WIN_SHELL=pwsh
# Interpreter used on local and remote hosts in Linux
# Available values: sh/bash/zsh or other
LINUX_SHELL=bash

# Parallel (async) execution of commands (default: false)
PARALLEL_EXEC=true

# Global parameters for ssh connection (low priority)
SSH_PORT=2121
SSH_USER=lifailon
# Use password to connect (optional)
SSH_PASSWORD=
# Full path to private key (default: ~/.ssh/id_rsa)
SSH_PRIVATE_KEY_PATH=
SSH_CONNECT_TIMEOUT=2
# Save and reuse passed variables and functions (default: false)
SSH_SAVE_ENV=true
# List of hosts separated by comma (high priority for username and port)
SSH_HOST_LIST=root@192.168.3.102:22,root@192.168.3.103:22,192.168.3.105,192.168.3.106
```

> [!NOTE]
> Access to the bot is limited by user ID. You can find out the Telegram `id` using [@getmyid_bot](https://t.me/getmyid_bot) or in the bot logs when sending a message to it.

- Run the bot in a container:

```shell
docker run -d --name ssh-bot \
    -v ./.env:/ssh-bot/.env \
    -v $HOME/.ssh/id_rsa:/root/.ssh/id_rsa \
    --restart unless-stopped \
    lifailon/ssh-bot:latest
```

> [!NOTE]
> To access remote hosts, authorization is used by a key (in the example, `.ssh/id_rsa`), which must be forwarded to the container from the host system.

## Build

```shell
git clone https://github.com/Lifailon/ssh-bot
cd ssh-bot
cp .env.example .env
docker-compose up -d --build
```