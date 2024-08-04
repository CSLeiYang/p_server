package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"math/rand"

	"csleiyang.com/p_server/dbutil"
	log "csleiyang.com/p_server/logger"
	"github.com/gorilla/mux"
)

type Translation struct {
	SourceLang string `json:"sourceLanguage"`
	TargetLang string `json:"targetLanguage"`
}
type MyConfig struct {
	TTSVoice string `json:"ttsVoice"`
	TTSSpeed string `json:"ttsSpeed"`
}
type User struct {
	ID         int        `json:"id"`
	Name       *string     `json:"name"`
	UserID     string     `json:"userid"`
	Password   string     `json:"password"`
	Account    int        `json:"account"`
	Age        int        `json:"age"`
	Purpose    string     `json:"purpose"`
	Created    time.Time  `json:"created"`
	Updated    *time.Time `json:"updated"`
	RewardInfo string     `json:"reward_info"`
	MyConfig   MyConfig   `json:"myconfig"`
	DeviceIDs  string     `json:"device_ids"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func RegisterUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	myConfigStr, err := json.Marshal(user.MyConfig)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("myConfigStr:", string(myConfigStr))

	query := "INSERT INTO user (name, userid, password, age, purpose, reward_info, myconfig, device_ids) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = db.Insert(query, user.Name, user.UserID, user.Password, user.Age, user.Purpose, user.RewardInfo, string(myConfigStr), user.DeviceIDs)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: user}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func LoginUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	log.Info("LoginUser")
	var user User
	var myConfigStr string
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "SELECT iduser, name, account, age, purpose, created, updated, reward_info, myconfig, device_ids FROM user WHERE userid = ? AND password = ?"
	row := db.QueryRow(query, user.UserID, user.Password)
	err = row.Scan(&user.ID, &user.Name, &user.Account, &user.Age, &user.Purpose, &user.Created, &user.Updated, &user.RewardInfo, &myConfigStr, &user.DeviceIDs)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: "Invalid credentials"}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}
	err = json.Unmarshal([]byte(myConfigStr), &user.MyConfig)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: "Invalid myConfig"}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: user}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func GenerateRandomInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return min + rand.Intn(max-min+1)
}
func GetUserHttp(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]

	query := "SELECT iduser, userid, name, account, age, purpose, created, updated, reward_info, myconfig, device_ids FROM user WHERE userid = ? or device_ids= ?"
	row := db.QueryRow(query, userID, userID)

	var user User
	var myConfigStr string
	err := row.Scan(&user.ID, &user.UserID, &user.Name, &user.Account, &user.Age, &user.Purpose, &user.Created, &user.Updated, &user.RewardInfo, &myConfigStr, &user.DeviceIDs)
	if err != nil {
		_, errNum := strconv.Atoi(userID)
		if err == sql.ErrNoRows && errNum != nil {
			//create new user based device_ids
			newUserId := GenerateRandomInt(1000000000, 9999999999)
			user.UserID = fmt.Sprintf("%d", newUserId)
			user.DeviceIDs = userID
			newQuery := "INSERT INTO user (userid, device_ids) VALUES (?, ?)"
			_, err = db.Insert(newQuery, newUserId, userID)
			if err == nil {

				response := Response{Success: true, Data: user}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return

			} else {
				response := Response{Success: false, Error: "Create User failed"}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return

			}

		} else {
			log.Error(err)
			response := Response{Success: false, Error: "User not found"}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

	}
	err = json.Unmarshal([]byte(myConfigStr), &user.MyConfig)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: "Invalid myConfig"}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: user}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func GetUser(db *dbutil.MySQLDB, deviceId string) (User, error) {
	query := "SELECT iduser, name, account, age, purpose, created, updated, reward_info, myconfig, device_ids FROM user WHERE userid = ?"
	row := db.QueryRow(query, deviceId)

	var user User
	var myConfigStr string
	err := row.Scan(&user.ID, &user.Name, &user.Account, &user.Age, &user.Purpose, &user.Created, &user.Updated, &user.RewardInfo, &myConfigStr, &user.DeviceIDs)
	if err != nil {
		return user, err
	}
	err = json.Unmarshal([]byte(myConfigStr), &user.MyConfig)
	if err != nil {
		log.Error(err)
	}
	return user, err
}

func UpdateUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	log.Info("user.MyConfig:", user.MyConfig)
	myConfigStr, err := json.Marshal(user.MyConfig)
	if err != nil {
		log.Error(err)
	}

	query := "UPDATE user SET name = ?, password = ?, age = ?, purpose = ?, reward_info = ?, myconfig = ?, device_ids = ?, updated = CURRENT_TIMESTAMP WHERE userid = ?"
	_, err = db.Update(query, user.Name, user.Password, user.Age, user.Purpose, user.RewardInfo, myConfigStr, user.DeviceIDs, userID)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func DeleteUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]

	query := "DELETE FROM user WHERE userid = ?"
	_, err := db.Delete(query, userID)
	if err != nil {
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
