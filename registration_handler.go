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

// 0 (user asked about bus number)
// 1 (user asked about bus stop number)
// 2 (user asked about which days, can self loop)
// 3 (user asked about what time)
type UserState struct {
	State int
	BusInfoJob
	ScheduledTime
	time.Weekday
}

func handleRegistration(message *tgbotapi.Message) tgbotapi.MessageConfig {

	chatID := message.Chat.ID
	storedUserState := getUserState(chatID)

	// If user is new
	if storedUserState == nil {
		userState := UserState{State: 0}
		saveUserState(chatID, userState)
		reply := tgbotapi.NewMessage(message.Chat.ID, "Which bus would you like to be alerted for?")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}
	if storedUserState.State == 0 {
		// TODO: Validate bus number
		storedUserState.BusServiceNo = message.Text
		storedUserState.State = 1
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(message.Chat.ID, "Which bus stop?")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}
	if storedUserState.State == 1 {
		// TODO: Validate bus stop number
		storedUserState.BusStopCode = message.Text
		storedUserState.State = 2
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(message.Chat.ID, "Which day?")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}
	if storedUserState.State == 2 {
		// TODO: Validate day
		dayInt, _ := strconv.Atoi(message.Text)
		storedUserState.Weekday = time.Weekday(dayInt)
		storedUserState.State = 3
		saveUserState(chatID, *storedUserState)
		reply := tgbotapi.NewMessage(message.Chat.ID, "What time? In (hh:mm)")
		reply.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		return reply
	}
	if storedUserState.State == 3 {
		// TODO: Validate time
		textArr := strings.Split(message.Text, ":")
		hour, _ := strconv.Atoi(textArr[0])
		minute, _ := strconv.Atoi(textArr[1])
		storedUserState.ScheduledTime = ScheduledTime{Hour: hour, Minute: minute}
		busInfoJob, dayToExecute, timeToExecute := storedUserState.BusInfoJob, storedUserState.Weekday, storedUserState.ScheduledTime
		addJob(busInfoJob, dayToExecute, timeToExecute)
		if dayToExecute == time.Now().Weekday() {
			addJobToTodayCronner(cronner, busInfoJob, timeToExecute)
		}
		replyMessage := fmt.Sprintf("You will be reminded for bus %s at bus stop %s every %s %02d:%02d", busInfoJob.BusServiceNo, busInfoJob.BusStopCode, dayToExecute.String(), timeToExecute.Hour, timeToExecute.Minute)
		reply := tgbotapi.NewMessage(message.Chat.ID, replyMessage)
		reply.ReplyToMessageID = message.MessageID
		deleteUserState(chatID)
		return reply
	}
	log.Fatal("Unhandled state reached")
	return tgbotapi.NewMessage(chatID, "This should not be received")
}

func getUserState(chatID int64) *UserState {
	key := []byte(strconv.FormatInt(chatID, 10))
	var storedUserState UserState

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

func saveUserState(chatID int64, userState UserState) {
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

func mockInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("1.com", "http://1.com"),
			tgbotapi.NewInlineKeyboardButtonSwitch("2sw", "open 2"),
			tgbotapi.NewInlineKeyboardButtonData("3", "3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("4", "4"),
			tgbotapi.NewInlineKeyboardButtonData("5", "5"),
			tgbotapi.NewInlineKeyboardButtonData("6", "6"),
		),
	)
	return numericKeyboard
}
