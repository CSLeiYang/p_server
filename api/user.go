package api

import (
	"encoding/json"
	"net/http"
	"time"

	"csleiyang.com/p_server/dbutil"
	log "csleiyang.com/p_server/logger"
	"github.com/gorilla/mux"
)

type UserMyConfig struct {
	TTSVoice   string `json:"tts_voice"`
	TTSSpeed   string `json:"tts_speed"`
	SourceLang string `json:"source_language"`
	TargetLang string `json:"target_language"`
}
type User struct {
	ID         int          `json:"id"`
	Name       string       `json:"name"`
	UserID     string       `json:"userid"`
	Password   string       `json:"password"`
	Account    int          `json:"account"`
	Age        int          `json:"age"`
	Purpose    string       `json:"purpose"`
	Created    time.Time    `json:"created"`
	Updated    *time.Time   `json:"updated"`
	RewardInfo string       `json:"reward_info"`
	MyConfig   UserMyConfig `json:"myconfig"`
	DeviceIDs  string       `json:"device_ids"`
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

	query := "INSERT INTO user (name, userid, password, age, purpose, reward_info, myconfig, device_ids) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = db.Insert(query, user.Name, user.UserID, user.Password, user.Age, user.Purpose, user.RewardInfo, user.MyConfig, user.DeviceIDs)
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
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "SELECT iduser, name, account, age, purpose, created, updated, reward_info, myconfig, device_ids FROM user WHERE userid = ? AND password = ?"
	row := db.QueryRow(query, user.UserID, user.Password)
	err = row.Scan(&user.ID, &user.Name, &user.Account, &user.Age, &user.Purpose, &user.Created, &user.Updated, &user.RewardInfo, &user.MyConfig, &user.DeviceIDs)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: "Invalid credentials"}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: user}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]

	query := "SELECT iduser, name, account, age, purpose, created, updated, reward_info, myconfig, device_ids FROM user WHERE userid = ?"
	row := db.QueryRow(query, userID)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Account, &user.Age, &user.Purpose, &user.Created, &user.Updated, &user.RewardInfo, &user.MyConfig, &user.DeviceIDs)
	if err != nil {
		response := Response{Success: false, Error: "User not found"}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: user}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateUser(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "UPDATE user SET name = ?, password = ?, age = ?, purpose = ?, reward_info = ?, myconfig = ?, device_ids = ?, updated = CURRENT_TIMESTAMP WHERE userid = ?"
	_, err = db.Update(query, user.Name, user.Password, user.Age, user.Purpose, user.RewardInfo, user.MyConfig, user.DeviceIDs, userID)
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
