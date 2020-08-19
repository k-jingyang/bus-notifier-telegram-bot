SG Bus Notifier - Telegram Bot ![alt text](https://github.com/k-jingyang/smellybus-telegram-bot/workflows/Go/badge.svg "Build status")
=====

Telegram bot that allows users to register for push notifications for Singapore bus arrival timings at specific times. 

Using this bot, users need not look up for bus arrival timings if it's a routine (e.g. the morning commute) 

**Think alarm clock but with bus arrival timings.**
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
> bus-notifier will look for `refdata/bus_services.txt` & `refdata/refdata.db` during execution. Ensure that these files are present before running. 

## Improvements

- [ ] Create a Makefile to download reference data, build binary and place them into a build folder

