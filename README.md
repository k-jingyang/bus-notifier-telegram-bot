## Bus - Telegram Bot 
![alt text](https://github.com/k-jingyang/smellybus-telegram-bot/workflows/Go/badge.svg "Build status")


## Setup
1. Create a .env file with the contents, place it in the root directory of the project
```
TELEGRAM_API_TOKEN=YOUR_TELEGRAM_API_TOKEN
LTA_API_TOKEN=YOUR_LTA_API_TOKEN
```
2. Generate reference data
```
$ cd refdata
$ chmod +x bootstrap_reference_data.sh
$ ./bootstrap_reference_data.sh
```
3. Run using `go run` or build binary using `go build`


## Improvements

- [ ] Create a Makefile to to download reference data, build binary and place them into a build folder

