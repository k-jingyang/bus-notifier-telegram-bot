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
	Days []time.Weekday
}

func handleRegistration(update tgbotapi.Update) tgbotapi.MessageConfig {

	// Handles inline keyboard when asked for day, this should replace line 88-97
	if update.CallbackQuery != nil {
		chatID := update.CallbackQuery.Message.Chat.ID
		storedUserState := getUserState(chatID)
		if storedUserState.State != 3 {
			log.Fatal("Get inline query when not asked about day")
		}
		dayInt, _ := strconv.Atoi(update.CallbackQuery.Data)
		storedUserState.Days = append(storedUserState.Days, time.Weekday(dayInt))
		storedUserState.State = 4
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "What time? In the format of hh:mm \n\n Stop me with /exit")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}

	message := update.Message
	chatID := message.Chat.ID

	// Exits the registration process
	if message.IsCommand() && message.Command() == "exit" {
		deleteUserState(chatID)
		reply := tgbotapi.NewMessage(message.Chat.ID, "Okay")
		return reply
	}

	storedUserState := getUserState(chatID)

	// If user is new
	if storedUserState == nil {
		if message.IsCommand() && message.Command() == "register" {
			userState := userState{State: 1}
			saveUserState(chatID, userState)
			reply := tgbotapi.NewMessage(chatID, "Which bus would you like to be alerted for?")
			return reply
		} else {
			reply := tgbotapi.NewMessage(chatID, "Start by sending me /register")
			return reply
		}
	}
	if storedUserState.State == 1 {
		// TODO: Validate bus number
		storedUserState.BusServiceNo = message.Text
		storedUserState.State = 2
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "Which bus stop? \n\n Stop me with /exit")
		return reply
	}
	if storedUserState.State == 2 {
		// TODO: Validate bus stop number, and check if said bus number exists in this bus stop
		storedUserState.BusStopCode = message.Text
		storedUserState.State = 3
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "Which day? \n\n Stop me with /exit")
		reply.ReplyMarkup = buildWeekdayKeyboard()
		return reply
	}
	if storedUserState.State == 3 {
		// TODO: Validate day
		dayInt, _ := strconv.Atoi(message.Text)
		storedUserState.Days = append(storedUserState.Days, time.Weekday(dayInt))
		storedUserState.State = 4
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(chatID, "What time? In the format of hh:mm \n\n Stop me with /exit")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}
	if storedUserState.State == 4 {
		// TODO: Validate time
		textArr := strings.Split(message.Text, ":")
		hour, _ := strconv.Atoi(textArr[0])
		minute, _ := strconv.Atoi(textArr[1])
		storedUserState.scheduledTime = scheduledTime{Hour: hour, Minute: minute}
		for _, day := range storedUserState.Days {
			busInfoJob, dayToExecute, timeToExecute := storedUserState.busInfoJob, day, storedUserState.scheduledTime
			addJob(busInfoJob, dayToExecute, timeToExecute)
			if dayToExecute == time.Now().Weekday() {
				addJobToTodayCronner(cronner, busInfoJob, timeToExecute)
			}
		}

		replyMessage := fmt.Sprintf("You will be reminded for bus %s at bus stop %s every %s %02d:%02d", storedUserState.BusServiceNo, storedUserState.BusStopCode, storedUserState.Days, storedUserState.Hour, storedUserState.Minute)
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
		log.Fatal(err)
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
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("states"))
		if err != nil {
			log.Fatal(err)
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
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("states"))
		if b == nil {
			log.Fatal("Bucket should exist but doesn't exist")
		}
		b.Delete(key)
		return nil
	})
}

func buildLocationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButtonLocation("Get nearby bus stops")})
}

func buildWeekdayKeyboard() tgbotapi.InlineKeyboardMarkup {
	var weekdayKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Mon", "1"),
			tgbotapi.NewInlineKeyboardButtonData("Tues", "2"),
			tgbotapi.NewInlineKeyboardButtonData("Wed", "3"),
			tgbotapi.NewInlineKeyboardButtonData("Thur", "4"),
			tgbotapi.NewInlineKeyboardButtonData("Fri", "5"),
			tgbotapi.NewInlineKeyboardButtonData("Sat", "6"),
			tgbotapi.NewInlineKeyboardButtonData("Sun", "0"),
			tgbotapi.NewInlineKeyboardButtonData("Done", "-1"),
		),
	)
	return weekdayKeyboard
}
