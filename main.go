package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "csleiyang.com/p_server/logger"

	private_channel "csleiyang.com/p_server/private_channel"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]*private_channel.PUdpConn)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

type Message struct {
	UserID  string                   `json:"userid"`
	Message string                   `json:"message"`
	BizInfo private_channel.PCommand `json:"biz_info"`
}

func main() {
	router := mux.NewRouter()

	// WebSocket route
	router.HandleFunc("/ws/{userid}", handleConnections)

	// HTTP routes
	router.HandleFunc("/upload/{userid}", uploadFile).Methods("POST")
	router.HandleFunc("/download/{userid}/{filename}", downloadFile).Methods("GET")
	router.HandleFunc("/public/download/{filename}", downloadPublicFile).Methods("GET")

	// File server for public images
	fs := http.FileServer(http.Dir("./public"))
	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", fs))

	// Add CORS support
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	// Start the server
	go handleMessages()

	log.Info("Server started on :8000")
	err := http.ListenAndServe(":8000", handler)
	if err != nil {
		log.Error("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userid"]

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Infof("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	udpAddr, err := net.ResolveUDPAddr("udp", "165.227.25.123:8099")
	if err != nil {
		log.Errorf("Failed to resolve UDP address: %v", err)
		return
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Errorf("Failed to dial UDP address: %v", err)
		return
	}

	userIdU, _ := strconv.ParseUint(userID, 10, 64)
	pUdpConn := private_channel.NewPudpConn(userIdU, udpConn, udpAddr, ws, private_channel.HandlePEvent)

	mutex.Lock()
	clients[ws] = pUdpConn
	mutex.Unlock()

	go handlePings(ws)

	defer func() {
		mutex.Lock()
		delete(clients, ws)
		mutex.Unlock()
		pUdpConn.UdpConnStop()
	}()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Infof("WebSocket read error: %v, msg:%v", err, msg)
			break
		}
		msg.UserID = userID
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		mutex.Lock()
		for ws, pUdpConn := range clients {
			msgUserIdUint, err := strconv.ParseUint(msg.UserID, 10, 64)
			if err != nil {
				log.Warnf("Can not parseUint,msg.UserID: %v", msg)
				break
			}

			if clients[ws].StreamId == msgUserIdUint {
				bizInfo, err := json.Marshal(&msg.BizInfo)
				if err != nil {
					log.Warn(err)
					continue
				}
				err = pUdpConn.SendPrivateMessage(&private_channel.PrivateMessage{
					StreamId:   msgUserIdUint,
					BizInfo:    string(bizInfo),
					Content:    []byte(msg.Message),
					IsReliable: true,
					IsStreaming: false,
				})
				if err != nil {
					log.Infof("UDP send error: %v", err)
					ws.Close()
					delete(clients, ws)
				}
			}
		}
		mutex.Unlock()
	}
}

func handlePings(ws *websocket.Conn) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := ws.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Infof("WebSocket ping error: %v", err)
				ws.Close()
				return
			}
		}
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userid"]
	log.Info("uploadFile comming")

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Infof("ParseMultipartForm error: %v", err)
		http.Error(w, "Failed to parse form", http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Infof("FormFile error: %v", err)
		http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	userUploadDir := filepath.Join("./user_data", userID, "upload")
	if _, err := os.Stat(userUploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(userUploadDir, 0755)
		if err != nil {
			log.Infof("Directory creation error: %v", err)
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}
	}

	dst, err := os.Create(filepath.Join(userUploadDir, handler.Filename))
	if err != nil {
		log.Infof("File creation error: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Infof("File copy error: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Successfully uploaded file: %s", handler.Filename)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userid"]
	filename := vars["filename"]

	userDownloadDir := filepath.Join("./user_data", userID, "download")
	filepath := filepath.Join(userDownloadDir, filename)
	file, err := os.Open(filepath)
	if err != nil {
		log.Infof("File open error: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = io.Copy(w, file)
	if err != nil {
		log.Infof("File copy error: %v", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
}

func downloadPublicFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	publicDir := "./public"
	filepath := filepath.Join(publicDir, filename)
	file, err := os.Open(filepath)
	if err != nil {
		log.Infof("File open error: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = io.Copy(w, file)
	if err != nil {
		log.Infof("File copy error: %v", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
}
