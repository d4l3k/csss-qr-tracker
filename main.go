package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var db *bolt.DB

func initDB() (func() error, error) {
	var err error
	db, err = bolt.Open("tickets.db", 0600, nil)
	if err != nil {
		return db.Close, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{"tickets"} {
			_, err = tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return db.Close, err
	}
	return db.Close, nil
}

func main() {
	flag.Parse()

	done, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer done()

	ro := mux.NewRouter()
	ro.Path("/api/genTickets").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ids []string
		json.NewDecoder(r.Body).Decode(&ids)
		log.Printf("%#v", ids)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("tickets"))
			for _, id := range ids {
				if err := bucket.Put([]byte(id), []byte("0")); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(true)
	})
	ro.Path("/api/checkin").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := ioutil.ReadAll(r.Body)
		log.Println("id", id)
		w.Header().Set("Content-Type", "application/json")
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("tickets"))
			bId := []byte(id)
			time := string(bucket.Get(bId))
			json.NewEncoder(w).Encode(time)
			if time == "0" {
				if err := bucket.Put(bId, []byte("1")); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
		}
	})
	ro.Path("/api/beer").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := ioutil.ReadAll(r.Body)
		log.Println("id", id)
		w.Header().Set("Content-Type", "application/json")
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("tickets"))
			bId := []byte(id)
			t := string(bucket.Get(bId))
			json.NewEncoder(w).Encode(t)
			if len(t) > 0 {
				timeInt, _ := strconv.ParseInt(t, 10, 64)
				tTime := time.Unix(timeInt, 0)
				if tTime.Add(1 * time.Hour).Before(time.Now()) {
					t = strconv.FormatInt(time.Now().Unix(), 10)
					if err := bucket.Put(bId, []byte(t)); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
		}
	})
	ro.PathPrefix("/static/").Handler(http.FileServer(http.Dir("./static")))
	ro.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})
	http.Handle("/", ro)
	log.Println("Listening 0.0.0.0:8282")
	log.Fatal(http.ListenAndServe("0.0.0.0:8282", nil))
}
