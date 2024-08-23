package api

import (
	"encoding/json"
	"net/http"

	"csleiyang.com/p_server/dbutil"
	log "csleiyang.com/p_server/logger"
)

func PostFeedBack(db *dbutil.MySQLDB, w http.ResponseWriter, r *http.Request) {
	var feedback string
	err := json.NewDecoder(r.Body).Decode(&feedback)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Info("feedback:", string(feedback))

	query := "INSERT INTO feedback (content) VALUES (?)"
	_, err = db.Insert(query, feedback)
	if err != nil {
		log.Error(err)
		response := Response{Success: false, Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{Success: true, Data: feedback}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
