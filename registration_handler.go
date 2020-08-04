package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// UserState stores the state of user's registration
// 0 (nothing)
// 1 (user asked about bus number)
// 2 (user asked about bus stop number)
// 3 (user asked about which days, can self loop)
// 4 (user asked about what time)
// 5 (user asked which alarm to delete)
type UserState struct {
	State int
	BusInfoJob
	SelectedDays map[time.Weekday]bool
}

func (userState *UserState) toggleDay(day time.Weekday) {
	userState.SelectedDays[day] = !userState.SelectedDays[day]
}

func (userState *UserState) getSelectedDays() []time.Weekday {
	selectedDays := []time.Weekday{}
	for k, v := range userState.SelectedDays {
		if v {
			selectedDays = append(selectedDays, k)
		}
	}
	sort.Slice(selectedDays, func(i, j int) bool {
		return int(selectedDays[i]) < int(selectedDays[j])
	})

	return selectedDays
}

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
		DeleteUserState(chatID)
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
		SaveUserState(chatID, userState)

		reply := tgbotapi.NewMessage(chatID, stringBuilder.String())
		return registrationReply{replyMessage: reply}
	}

	storedUserState := GetUserState(chatID)

	// If db does not have this record
	if storedUserState == nil {
		if message != nil && message.IsCommand() && message.Command() == "register" {
			userState := UserState{State: 1, SelectedDays: make(map[time.Weekday]bool)}
			SaveUserState(chatID, userState)
			reply := tgbotapi.NewMessage(chatID, "Which bus would you like to be alerted for?")
			return registrationReply{replyMessage: reply}
		}
		reply := tgbotapi.NewMessage(chatID, "Start by sending me /register or if you want to delete an alarm, send me /delete")
		return registrationReply{replyMessage: reply}
	}

	switch storedUserState.State {
	case 1:
		if busServiceLookUp[message.Text] {
			busServiceNo := message.Text
			storedUserState.BusServiceNo = busServiceNo
			storedUserState.State = 2
			SaveUserState(chatID, *storedUserState)
			reply := tgbotapi.NewMessage(chatID, "Which bus stop? \n\nStop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		reply := tgbotapi.NewMessage(chatID, "Invalid bus, please try again \n\nStop me with /exit")
		return registrationReply{replyMessage: reply}
	case 2:
		// TODO: Validate bus stop number, and check if said bus number exists in this bus stop
		storedUserState.BusStopCode = message.Text
		storedUserState.State = 3
		SaveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "Which days? \n\nStop me with /exit")
		reply.ReplyMarkup = buildWeekdayKeyboard()
		return registrationReply{replyMessage: reply}
	case 3:
		if update.CallbackQuery != nil {
			dayInt, _ := strconv.Atoi(update.CallbackQuery.Data)
			// If user doesn't click on Done, store day
			if dayInt != -1 {
				storedUserState.toggleDay(time.Weekday(dayInt))
				SaveUserState(chatID, *storedUserState)

				stringBuilder := strings.Builder{}
				stringBuilder.WriteString("Which days? \nSelected: ")
				if len(storedUserState.getSelectedDays()) == 0 {
					stringBuilder.WriteString("None")
				} else {
					selectedDays := storedUserState.getSelectedDays()
					stringBuilder.WriteString(joinDaysString(selectedDays))
				}
				stringBuilder.WriteString("\nStop me with /exit")

				messageID := update.CallbackQuery.Message.MessageID
				editedMessage := tgbotapi.NewEditMessageText(chatID, messageID, stringBuilder.String())
				editedMessage.ReplyMarkup = buildWeekdayKeyboard()

				// Need to send CallBackConfig back, so that button stops the loading animation
				callBackID := update.CallbackQuery.ID
				return registrationReply{replyMessage: editedMessage, callbackResponse: tgbotapi.NewCallback(callBackID, "")}
			}
			storedUserState.State = 4
			SaveUserState(chatID, *storedUserState)
			reply := tgbotapi.NewMessage(chatID, "What time? In the format of hh:mm \n\nStop me with /exit")
			reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
			return registrationReply{replyMessage: reply}
		}
	case 4:
		textArr := strings.Split(message.Text, ":")
		hour, err := strconv.Atoi(textArr[0])
		if err != nil || hour > 23 {
			reply := tgbotapi.NewMessage(chatID, "Invalid time specified. In the format of hh:mm please.\n Stop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		minute, err := strconv.Atoi(textArr[1])
		if err != nil || minute > 59 {
			reply := tgbotapi.NewMessage(chatID, "Invalid time specified. In the format of hh:mm please.\n Stop me with /exit")
			return registrationReply{replyMessage: reply}
		}
		storedUserState.ScheduledTime = ScheduledTime{Hour: hour, Minute: minute}
		for _, day := range storedUserState.getSelectedDays() {
			dailyBusInfoJob := storedUserState.BusInfoJob
			dailyBusInfoJob.Weekday = day
			storedJobDB.StoreJob(dailyBusInfoJob)
			if day == time.Now().Weekday() {
				addJobToTodayCronner(todayCronner, dailyBusInfoJob)
			}
		}

		replyMessage := fmt.Sprintf("You will be reminded for bus %s at bus stop %s every %s %02d:%02d",
			storedUserState.BusServiceNo,
			storedUserState.BusStopCode,
			joinDaysString(storedUserState.getSelectedDays()),
			storedUserState.ScheduledTime.Hour,
			storedUserState.ScheduledTime.Minute)
		reply := tgbotapi.NewMessage(chatID, replyMessage)
		reply.ReplyToMessageID = message.MessageID
		DeleteUserState(chatID)
		return registrationReply{replyMessage: reply}
	case 5:
		selectedIndex, err := strconv.Atoi(message.Text)
		indexToDelete := selectedIndex - 1
		storedJobs := storedJobDB.GetJobsByChatID(chatID)

		if err != nil || indexToDelete < 0 || indexToDelete >= len(storedJobs) {
			reply := tgbotapi.NewMessage(chatID, "Invalid selection\n Stop me with /exit")
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

	log.Fatalln("Unhandled state reached")
	return registrationReply{replyMessage: tgbotapi.NewMessage(chatID, "Unexpected error has occured")}
}

// GetUserState retrieves the stored user state
func GetUserState(chatID int64) *UserState {
	key := []byte(strconv.FormatInt(chatID, 10))
	var storedUserState UserState

	db, err := bolt.Open("user_state.db", 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("states"))
		if b == nil {
			return errors.New("Bucket does not exist")
		}
		storedValue := b.Get(key)
		if storedValue == nil {
			return errors.New("Key does not exist")
		}
		json.Unmarshal(storedValue, &storedUserState)
		return nil
	})

	// If there's no matching record in database
	if err != nil {
		return nil
	}
	return &storedUserState
}

// SaveUserState saves the user state, holding information about the stage of registration that the user is at
func SaveUserState(chatID int64, userState UserState) {
	userState.ChatID = chatID
	log.Println("Saving user interaction state:", userState)

	key := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open("user_state.db", 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("states"))
		if err != nil {
			log.Fatalln(err)
		}

		encUserState, err := json.Marshal(userState)
		b.Put(key, encUserState)
		return nil
	})
}

// DeleteUserState deletes the saved user state
func DeleteUserState(chatID int64) {
	key := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open("user_state.db", 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("states"))
		if b == nil {
			log.Fatalln("Bucket should exist but doesn't exist")
		}
		b.Delete(key)
		return nil
	})
}

func buildLocationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButtonLocation("Get nearby bus stops")})
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
