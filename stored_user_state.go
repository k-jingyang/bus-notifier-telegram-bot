package main

import (
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
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

// ToggleDay toggles the truthy selection of the day
func (userState *UserState) ToggleDay(day time.Weekday) {
	userState.SelectedDays[day] = !userState.SelectedDays[day]
}

// GetSelectedDays gets a list of selected days
func (userState *UserState) GetSelectedDays() []time.Weekday {
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

// UserStateDB contains the operations to store/retrieve/delete user states,
// holding information about the stage of registration that the user is at
type UserStateDB struct {
	dbFile       string
	statesBucket string
}

// NewUserStateDB returns an initialised instance of UserStateDB
func NewUserStateDB(dbFile string) UserStateDB {
	return UserStateDB{dbFile: dbFile, statesBucket: "users"}
}

// GetUserState retrieves the stored user state
func (s *UserStateDB) GetUserState(chatID int64) *UserState {
	key := []byte(strconv.FormatInt(chatID, 10))
	var storedUserState UserState

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.statesBucket))
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

// SaveUserState saves the user state,
func (s *UserStateDB) SaveUserState(chatID int64, userState UserState) {
	userState.ChatID = chatID
	log.Println("Saving user interaction state:", userState)

	key := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(s.statesBucket))
		if err != nil {
			log.Fatalln(err)
		}

		encUserState, err := json.Marshal(userState)
		b.Put(key, encUserState)
		return nil
	})
}

// DeleteUserState deletes the saved user state
func (s *UserStateDB) DeleteUserState(chatID int64) {
	key := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.statesBucket))
		if b == nil {
			log.Fatalln("Bucket should exist but doesn't exist")
		}
		b.Delete(key)
		return nil
	})
}
