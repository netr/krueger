# Krueger

Krueger is designed to run alongside your VPN connection.
If for any reason your IP changes, while you are connected to a VPN,
all the processes that you've set will be shutdown immediately.
It uses a UDP connection to get your hostname IP every second.

For ease of use, it monitors the IP address you're connected to when you start Krueger.

![Screenshot of Krueger running](https://i.postimg.cc/zDdL7HJn/Screenshot-from-2022-11-29-22-56-35.png)]

## Config
~/.config/.krueger.yaml
```yaml
# ~/.config/.krueger.yaml

processes: brave,firefox,chrome,keybase,telegram,discord,aim,irc,icq # add as many process names here as you want
```

## Install
`go install github.com/netr/krueger@latest`

## Run
- `krueger` - Good to go.
- `krueger --debug` - Runs debug mode, which pretty prints a table of all your processes by name and PID.
- `krueger --help`