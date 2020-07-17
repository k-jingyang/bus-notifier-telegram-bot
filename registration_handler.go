package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// userState stores the state of user's registration
// 0 (nothing)
// 1 (user asked about bus number)
// 2 (user asked about bus stop number)
// 3 (user asked about which days, can self loop)
// 4 (user asked about what time)
type userState struct {
	State int
	busInfoJob
	scheduledTime
	SelectedDays map[time.Weekday]bool
}

func (userState *userState) toggleDay(day time.Weekday) {
	userState.SelectedDays[day] = !userState.SelectedDays[day]
}

func (userState *userState) getSelectedDays() []time.Weekday {
	selectedDays := []time.Weekday{}
	for k, v := range userState.SelectedDays {
		if v {
			selectedDays = append(selectedDays, k)
		}
	}
	return selectedDays
}

func handleRegistration(update tgbotapi.Update) tgbotapi.Chattable {

	if update.CallbackQuery == nil && update.Message == nil {
		log.Fatalln("Both cannot be nil at the same time")
	}

	var chatID int64
	if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		chatID = update.Message.Chat.ID
	}

	message := update.Message

	// Exits the registration process
	if message != nil && message.IsCommand() && message.Command() == "exit" {
		deleteUserState(chatID)
		reply := tgbotapi.NewMessage(message.Chat.ID, "Okay")
		return reply
	}

	storedUserState := getUserState(chatID)

	// If db does not have this record
	if storedUserState == nil {
		if message.IsCommand() && message.Command() == "register" {
			userState := userState{State: 1, SelectedDays: make(map[time.Weekday]bool)}
			saveUserState(chatID, userState)
			reply := tgbotapi.NewMessage(chatID, "Which bus would you like to be alerted for?")
			return reply
		} else {
			reply := tgbotapi.NewMessage(chatID, "Start by sending me /register")
			return reply
		}
	}

	switch storedUserState.State {
	case 1:
		if busServiceLookUp[message.Text] {
			busServiceNo := message.Text
			storedUserState.BusServiceNo = busServiceNo
			storedUserState.State = 2
			saveUserState(chatID, *storedUserState)
			reply := tgbotapi.NewMessage(chatID, "Which bus stop? \n\n Stop me with /exit")
			return reply
		} else {
			reply := tgbotapi.NewMessage(chatID, "Invalid bus, please try again \n\n Stop me with /exit")
			return reply
		}
	case 2:
		// TODO: Validate bus stop number, and check if said bus number exists in this bus stop
		storedUserState.BusStopCode = message.Text
		storedUserState.State = 3
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "Which day? \n\n Stop me with /exit")
		reply.ReplyMarkup = buildWeekdayKeyboard()
		return reply
	case 3:
		if update.CallbackQuery != nil {
			dayInt, _ := strconv.Atoi(update.CallbackQuery.Data)
			// If user doesn't click on Done, store day
			if dayInt != -1 {
				storedUserState.toggleDay(time.Weekday(dayInt))
				saveUserState(chatID, *storedUserState)

				stringBuilder := strings.Builder{}
				stringBuilder.WriteString("Which day? \n Selected: ")
				if len(storedUserState.getSelectedDays()) == 0 {
					stringBuilder.WriteString("None")
				} else {
					selectedDays := storedUserState.getSelectedDays()
					stringBuilder.WriteString(joinDaysString(selectedDays))
				}
				stringBuilder.WriteString("\n Stop me with /exit")

				messageID := update.CallbackQuery.Message.MessageID
				editedMessage := tgbotapi.NewEditMessageText(chatID, messageID, stringBuilder.String())
				editedMessage.ReplyMarkup = buildWeekdayKeyboard()

				// TODO: Need to send this back via AnswerCallbackQuery, so that button stops the loading
				// callBackID := update.CallbackQuery.ID
				// callbackQueryResponse := tgbotapi.NewCallback(callBackID, "")
				return editedMessage
			} else {
				storedUserState.State = 4
				saveUserState(chatID, *storedUserState)
				reply := tgbotapi.NewMessage(chatID, "What time? In the format of hh:mm \n\n Stop me with /exit")
				reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
				return reply
			}
		}
	case 4:
		// TODO: Validate time
		textArr := strings.Split(message.Text, ":")
		hour, _ := strconv.Atoi(textArr[0])
		minute, _ := strconv.Atoi(textArr[1])
		storedUserState.scheduledTime = scheduledTime{Hour: hour, Minute: minute}
		for _, day := range storedUserState.getSelectedDays() {
			busInfoJob, dayToExecute, timeToExecute := storedUserState.busInfoJob, day, storedUserState.scheduledTime
			addJob(busInfoJob, dayToExecute, timeToExecute)
			if dayToExecute == time.Now().Weekday() {
				addJobToTodayCronner(todayCronner, busInfoJob, timeToExecute)
			}
		}

		replyMessage := fmt.Sprintf("You will be reminded for bus %s at bus stop %s every %s %02d:%02d",
			storedUserState.BusServiceNo,
			storedUserState.BusStopCode,
			joinDaysString(storedUserState.getSelectedDays()),
			storedUserState.Hour,
			storedUserState.Minute)
		reply := tgbotapi.NewMessage(chatID, replyMessage)
		reply.ReplyToMessageID = message.MessageID
		deleteUserState(chatID)
		return reply
	}
	log.Println("Unhandled state reached")
	return tgbotapi.NewMessage(chatID, "This should not be received")
}

func getUserState(chatID int64) *userState {
	key := []byte(strconv.FormatInt(chatID, 10))
	var storedUserState userState

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

func saveUserState(chatID int64, userState userState) {
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

func deleteUserState(chatID int64) {
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
			tgbotapi.NewInlineKeyboardButtonData("Sat", strconv.Itoa(int(time.Saturday))),
			tgbotapi.NewInlineKeyboardButtonData("Sun", strconv.Itoa(int(time.Sunday))),
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
