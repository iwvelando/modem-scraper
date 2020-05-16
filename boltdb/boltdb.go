package boltdb

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pdunnavant/modem-scraper/config"
	"github.com/pdunnavant/modem-scraper/scrape"

	"github.com/boltdb/bolt"

	"github.com/OneOfOne/xxhash"
)

// PruneEventLogs queries BoltDB for matching logs and removes them from
// ModemInformation if found
func PruneEventLogs(config config.BoltDB, modemInformation scrape.ModemInformation) (*scrape.ModemInformation, error) {

	db, err := bolt.Open(config.Path, 0600, nil)
	if err != nil {
		fmt.Println(err)
		return &modemInformation, nil
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("EventLogs"))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	var newEventLog []scrape.EventLog
	for _, log := range modemInformation.EventLog {
		hash := HashLog(log)
		if !AlreadyLogged(db, log.DateTime, hash) {
			//fmt.Printf("Preserving element %s\n", hash)
			newEventLog = append(newEventLog, log)
		}
	}
	modemInformation.EventLog = newEventLog

	return &modemInformation, nil
}

// UpdateEventLogs queries BoltDB to write in the record of logs that have been
// successfully written to InfluxDB and/or MQTT so that we do not rewrite later
func UpdateEventLogs(config config.BoltDB, modemInformation scrape.ModemInformation) error {

	db, err := bolt.Open(config.Path, 0600, nil)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("EventLogs"))
		if err != nil {
			return err
		}
		return nil
	})

	hashMap := ArrangeHashes(modemInformation.EventLog)
	hashMap = AppendFromExisting(db, hashMap)

	db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("EventLogs"))
		for dateTime, hashes := range hashMap {
			hashesJson, err := json.Marshal(hashes)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = b.Put([]byte(dateTime), hashesJson)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil
}

func GetUniqueDateTimes(e []scrape.EventLog) []string {
	dateTimes := []string{}
	keys := make(map[string]bool)
	for _, entry := range e {
		if _, value := keys[entry.DateTime]; !value {
			keys[entry.DateTime] = true
			dateTimes = append(dateTimes, entry.DateTime)
		}
	}

	return dateTimes
}

func ArrangeHashes(e []scrape.EventLog) map[string][]string {
	dateTimes := GetUniqueDateTimes(e)
	hashMap := make(map[string][]string)

	for _, dateTime := range dateTimes {
		for _, log := range e {
			if log.DateTime == dateTime {
				hash := HashLog(log)
				if !ElementOf(hashMap[dateTime], hash) {
					hashMap[dateTime] = append(hashMap[dateTime], hash)
				}
			}
		}

	}

	return hashMap
}

func HashLog(log scrape.EventLog) string {
	logConcat := log.DateTime + strconv.Itoa(log.EventID) + strconv.Itoa(log.EventLevel) + log.Description
	logConcatHash := strconv.FormatUint(xxhash.Checksum64([]byte(logConcat)), 16)
	return logConcatHash
}

func AppendFromExisting(db *bolt.DB, hashMap map[string][]string) map[string][]string {

	var dbHashes []string
	for dateTime, hashes := range hashMap {
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("EventLogs"))
			v := b.Get([]byte(dateTime))
			if v != nil {
				err := json.Unmarshal(v, &dbHashes)
				if err != nil {
					fmt.Println(err)
					return err
				}
				//fmt.Printf("Initial hashes %v\n", dbHashes)
				for _, hash := range dbHashes {
					if !ElementOf(hashes, hash) {
						hashMap[dateTime] = append(hashMap[dateTime], hash)
					}
				}
				//fmt.Printf("Appended hashes %v\n", dbHashes)
			}
			return nil
		})
	}

	return hashMap
}

func AlreadyLogged(db *bolt.DB, dateTime string, hash string) bool {

	var dbHashes []string
	found := false
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("EventLogs"))
		v := b.Get([]byte(dateTime))
		if v != nil {
			err := json.Unmarshal(v, &dbHashes)
			if err != nil {
				fmt.Println(err)
				return err
			}
			//fmt.Println(len(dbHashes))
			if ElementOf(dbHashes, hash) {
				found = true
			}
		}
		return nil
	})

	return found
}

func ElementOf(slice []string, val string) bool {

	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

func PruneElement(slice []scrape.EventLog, index int) []scrape.EventLog {
	slice[index] = slice[len(slice)-1]
	slice[len(slice)-1] = scrape.EventLog{}
	slice = slice[:len(slice)-1]
	return slice
}
