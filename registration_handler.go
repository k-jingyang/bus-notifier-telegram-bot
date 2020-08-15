package main

import (
	"bus-notifier/refdata"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type registrationReply struct {
	replyMessage     tgbotapi.Chattable
	callbackResponse tgbotapi.CallbackConfig
}

func handleRegistration(update tgbotapi.Update) registrationReply {
	var chatID int64
	if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		chatID = update.Message.Chat.ID
	}

	message := update.Message

	// Exits the registration process
	if message != nil && message.IsCommand() && message.Command() == "exit" {
		userStateDB.DeleteUserState(chatID)
		reply := tgbotapi.NewMessage(chatID, "Okay!")
		return registrationReply{replyMessage: reply}
	}

	if message != nil && message.IsCommand() && message.Command() == "delete" {
		storedJobs := storedJobDB.GetJobsByChatID(chatID)
		if len(storedJobs) == 0 {
			reply := tgbotapi.NewMessage(chatID, "You have no registered alarms")
			return registrationReply{replyMessage: reply}
		}

		stringBuilder := strings.Builder{}
		stringBuilder.WriteString("Which alarm do you want to delete? Tell me the number!\n")
		for i, job := range storedJobs {
			jobString := fmt.Sprintf("%d. %s - %s - Bus %s @ %s", i+1, job.Weekday.String(), job.ScheduledTime.ToString(), job.BusServiceNo, job.BusStopCode)
			stringBuilder.WriteString(jobString)
			stringBuilder.WriteString("\n")
		}
		stringBuilder.WriteString("\nStop me with /exit")
		userState := UserState{State: 5, SelectedDays: make(map[time.Weekday]bool)}
		userStateDB.SaveUserState(chatID, userState)

		reply := tgbotapi.NewMessage(chatID, stringBuilder.String())
		return registrationReply{replyMessage: reply}
	}

	storedUserState := userStateDB.GetUserState(chatID)

	// If db does not have this record
	if storedUserState == nil {
		if message != nil && message.IsCommand() && message.Command() == "register" {
			userState := UserState{State: 1, SelectedDays: make(map[time.Weekday]bool)}
			userStateDB.SaveUserState(chatID, userState)
			reply := tgbotapi.NewMessage(chatID, "Which bus would you like to be alerted for?")
			return registrationReply{replyMessage: reply}
		}
		reply := tgbotapi.NewMessage(chatID, "Start by sending me /register or if you want to delete an alarm, send me /delete")
		return registrationReply{replyMessage: reply}
	}

	// Only state 3 should have nil message
	if storedUserState.State != 3 && message == nil {
		return registrationReply{replyMessage: tgbotapi.NewMessage(chatID, "I don't understand.")}
	}

	switch storedUserState.State {

	case 1:
		if busServiceLookUp[message.Text] {
			busServiceNo := message.Text
			storedUserState.BusServiceNo = busServiceNo
			storedUserState.State = 2
			userStateDB.SaveUserState(chatID, *storedUserState)

			transitLinkURL := fmt.Sprintf("https://www.transitlink.com.sg/eservice/eguide/service_route.php?service=%s", busServiceNo)
			message := fmt.Sprintf("Which bus stop do you want to be alerted for? Tell me the bus stop code. \n\nYou can look for the bus stop code at %s \n\nStop me with /exit", transitLinkURL)

			reply := tgbotapi.NewMessage(chatID, message)
			return registrationReply{replyMessage: reply}
		}
		reply := tgbotapi.NewMessage(chatID, "Invalid bus, please try again \n\nStop me with /exit")
		return registrationReply{replyMessage: reply}

	case 2:
		inputBusStopCode := message.Text
		for _, busRoute := range refDataDB.GetBusRoutesByBusService(storedUserState.BusServiceNo) {
			if busRoute.BusStopCode == inputBusStopCode {
				storedUserState.BusStopCode = inputBusStopCode
				storedUserState.State = 3
				userStateDB.SaveUserState(chatID, *storedUserState)
				reply := tgbotapi.NewMessage(chatID, "Which days? \n\nStop me with /exit")
				reply.ReplyMarkup = buildWeekdayKeyboard()
				return registrationReply{replyMessage: reply}
			}
		}
		transitLinkURL := fmt.Sprintf("https://www.transitlink.com.sg/eservice/eguide/service_route.php?service=%s", storedUserState.BusServiceNo)

		message := fmt.Sprintf("This bus stop is not serviced by the bus %s, please try again. \n\nYou can look for the bus stop code at %s, \n\nStop me with /exit", storedUserState.BusServiceNo, transitLinkURL)
		reply := tgbotapi.NewMessage(chatID, message)
		return registrationReply{replyMessage: reply}

	case 3:
		if update.CallbackQuery != nil {
			dayInt, _ := strconv.Atoi(update.CallbackQuery.Data)
			// If user doesn't click on Done, store day
			if dayInt != -1 {
				storedUserState.ToggleDay(time.Weekday(dayInt))
				userStateDB.SaveUserState(chatID, *storedUserState)

				stringBuilder := strings.Builder{}
				stringBuilder.WriteString("Which days? \nSelected: ")
				if len(storedUserState.GetSelectedDays()) == 0 {
					stringBuilder.WriteString("None")
				} else {
					selectedDays := storedUserState.GetSelectedDays()
					stringBuilder.WriteString(joinDaysString(selectedDays))
				}
				stringBuilder.WriteString("\n\nStop me with /exit")

				messageID := update.CallbackQuery.Message.MessageID
				editedMessage := tgbotapi.NewEditMessageText(chatID, messageID, stringBuilder.String())
				editedMessage.ReplyMarkup = buildWeekdayKeyboard()

				// Need to send CallBackConfig back, so that button stops the loading animation
				callBackID := update.CallbackQuery.ID
				return registrationReply{replyMessage: editedMessage, callbackResponse: tgbotapi.NewCallback(callBackID, "")}
			}
			storedUserState.State = 4
			userStateDB.SaveUserState(chatID, *storedUserState)
			reply := tgbotapi.NewMessage(chatID, "What time? In the format of hh:mm \n\nStop me with /exit")
			return registrationReply{replyMessage: reply}
		}

	case 4:
		textArr := strings.Split(message.Text, ":")
		hour, err := strconv.Atoi(textArr[0])
		if err != nil || hour > 23 {
			reply := tgbotapi.NewMessage(chatID, "Invalid time specified. In the format of hh:mm please.\n\nStop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		minute, err := strconv.Atoi(textArr[1])
		if err != nil || minute > 59 {
			reply := tgbotapi.NewMessage(chatID, "Invalid time specified. In the format of hh:mm please.\n\nStop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		storedUserState.ScheduledTime = ScheduledTime{Hour: hour, Minute: minute}
		for _, day := range storedUserState.GetSelectedDays() {
			dailyBusInfoJob := storedUserState.BusInfoJob
			dailyBusInfoJob.Weekday = day
			storedJobDB.StoreJob(dailyBusInfoJob)
			if day == time.Now().Weekday() {
				addJobtoCronner(cronner, dailyBusInfoJob)
			}
		}

		replyMessage := fmt.Sprintf("You will be reminded for bus %s at %s (%s) every %s %02d:%02d",
			storedUserState.BusServiceNo,
			refDataDB.GetBusStopByBusStopCode(storedUserState.BusStopCode).Description,
			storedUserState.BusStopCode,
			joinDaysString(storedUserState.GetSelectedDays()),
			storedUserState.ScheduledTime.Hour,
			storedUserState.ScheduledTime.Minute)
		reply := tgbotapi.NewMessage(chatID, replyMessage)
		reply.ReplyToMessageID = message.MessageID
		userStateDB.DeleteUserState(chatID)
		return registrationReply{replyMessage: reply}

	case 5:
		selectedIndex, err := strconv.Atoi(message.Text)
		indexToDelete := selectedIndex - 1
		storedJobs := storedJobDB.GetJobsByChatID(chatID)

		if err != nil || indexToDelete < 0 || indexToDelete >= len(storedJobs) {
			reply := tgbotapi.NewMessage(chatID, "Invalid selection\n\nStop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		storedJobDB.DeleteJob(storedJobs[indexToDelete])

		remainingJobs := storedJobDB.GetJobsByChatID(chatID)
		stringBuilder := strings.Builder{}
		stringBuilder.WriteString("Which alarm do you want to delete? Tell me the number!\n")
		for i, job := range remainingJobs {
			jobString := fmt.Sprintf("%d. %s - %s - Bus %s @ %s", i+1, job.Weekday.String(), job.ScheduledTime.ToString(), job.BusServiceNo, job.BusStopCode)
			stringBuilder.WriteString(jobString)
			stringBuilder.WriteString("\n")
		}
		stringBuilder.WriteString("\nStop me with /exit")

		reply := tgbotapi.NewMessage(chatID, stringBuilder.String())
		return registrationReply{replyMessage: reply}
	}
	return registrationReply{replyMessage: tgbotapi.NewMessage(chatID, "I don't understand.")}
}

func buildWeekdayKeyboard() *tgbotapi.InlineKeyboardMarkup {
	var weekdayKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Mon", strconv.Itoa(int(time.Monday))),
			tgbotapi.NewInlineKeyboardButtonData("Tues", strconv.Itoa(int(time.Tuesday))),
			tgbotapi.NewInlineKeyboardButtonData("Wed", strconv.Itoa(int(time.Wednesday))),
			tgbotapi.NewInlineKeyboardButtonData("Thur", strconv.Itoa(int(time.Thursday))),
			tgbotapi.NewInlineKeyboardButtonData("Fri", strconv.Itoa(int(time.Friday))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Sat", strconv.Itoa(int(time.Saturday))),
			tgbotapi.NewInlineKeyboardButtonData("Sun", strconv.Itoa(int(time.Sunday))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Done", "-1"),
		),
	)
	return &weekdayKeyboard
}

func joinDaysString(days []time.Weekday) string {
	stringBuilder := strings.Builder{}
	for i, day := range days {
		stringBuilder.WriteString(day.String())
		if i < len(days)-1 {
			stringBuilder.WriteString(", ")
		}
	}
	return stringBuilder.String()
}

// splitByDireciton returns bus stops serviced by a bus in direction 1 and direction 2
func splitByDirection(busRoutes []refdata.BusRoute) ([]refdata.BusRoute, []refdata.BusRoute) {
	var direction1 []refdata.BusRoute
	var direction2 []refdata.BusRoute

	for _, busRoute := range busRoutes {
		if busRoute.Direction == 1 {
			direction1 = append(direction1, busRoute)
		} else {
			direction2 = append(direction2, busRoute)
		}
	}
	return direction1, direction2
}
