<h1 align="center">
    <img src="img/logo.png" width="220" />
    <div>
    SSH Bot
    </div>
</h1>

<h4 align="center">
    <strong>English (🇺🇸)</strong> | <a href="README_RU.md">Русский (🇷🇺)</a>
</h4>

This is a Telegram bot that allows you to run specified commands on a remote machine and return the results of their execution. There is support for directory navigation and execution of commands on remote hosts via `ssh` without establishing a permanent connection.

The bot provides the opportunity not to waste time on setting up a `VPN` server and money on an external IP address or VPS server to access the local network, and also eliminates the need to use third-party applications (`VPN` and `ssh` clients) on a remote device and does not require a stable Internet connection.

## Roadmap

- [X] Executing commands on the local (where the bot is running) or remote host (via `ssh`) in the specified interpreter.
- [X] Support for parallel (asynchronous) command execution.
- [X] `ssh` connection manager with host availability check.
- [X] Support for directory navigation.
- [ ] Access to remote hosts by password or key from the configuration.
- [ ] Processing commands that require user input.
- [ ] Simulating a user session to store variables.

## Launch

You can download the precompiled executable from the [releases](https://github.com/Lifailon/ssh-bot/releases) page and run the bot locally (the [env](/pkg/env/env.go) package handles the parameters) or in a Docker container using an image from [Docker Hub](https://hub.docker.com/r/lifailon/ssh-bot).

> [!NOTE]
> Before launching, you need to create your Telegram bot using [@BotFather](https://telegram.me/BotFather) and get its `API Token`, which must be specified in the configuration file.

- Create a working directory:

```shell
mkdir ssh-bot
cd ssh-bot
```

- Create and fill the `.env` file:

```shell
TELEGRAM_BOT_TOKEN=XXXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
TELEGRAM_USER_ID=7777777777

# The interpreter used on the local host (Windows only)
# Available values: powershell/pwsh
WIN_SHELL=pwsh
# The interpreter to use on the local or remote host (Linux only)
# Available values: sh/bash or other
LINUX_SHELL=bash

# Parallel (async) execution of commands (default: false)
PARALLEL_EXEC=true

# Global parameters for ssh connection (low priority)
SSH_USER=root
SSH_PORT=22
SSH_CONNECT_TIMEOUT=2

# List of hosts separated by comma (high priority for username and port)
SSH_HOST_LIST=192.168.3.102,192.168.3.103,lifailon@192.168.3.105:2121
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

> [!WARNING]
> To access remote hosts, authorization is used by a key (`.ssh/id_rsa`), which must be forwarded to the container from the host system.

## Build

```shell
git clone https://github.com/Lifailon/ssh-bot
cd ssh-bot
docker-compose up -d --build
```