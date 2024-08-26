package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	api "csleiyang.com/p_server/api"
	dbutil "csleiyang.com/p_server/dbutil"
	log "csleiyang.com/p_server/logger"
	private_channel "csleiyang.com/p_server/private_channel"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	clients   = make(map[string]*private_channel.PUdpConn)
	broadcast = make(chan private_channel.WsMessage)
	mutex     sync.Mutex
	db        *dbutil.MySQLDB
)

//openssl genpkey -algorithm RSA -out private.key

//openssl req -new -x509 -key private.key -out certificate.crt -days 3650

func main() {
	var err error
	// dsn := "user:password@tcp(localhost:3306)/dbname"
	dsn := "test.db"
	db, err = dbutil.NewMySQLDB(dsn)
	if err != nil {
		log.Error(err)
	}
	defer db.Close()

	err = db.CreateTable()
	if err != nil {
		log.Error(err)
		return
	}

	router := mux.NewRouter()

	// WebSocket route
	router.HandleFunc("/ws/{userid}", handleConnections)

	// HTTP routes
	router.HandleFunc("/upload/{userid}", uploadFile).Methods("POST")
	router.HandleFunc("/download/{userid}/{filename}", downloadFile).Methods("GET")
	router.HandleFunc("/public/download/{filename}", downloadPublicFile).Methods("GET")

	// User routes
	router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		api.RegisterUser(db, w, r)
	}).Methods("POST")
	router.HandleFunc("/feedback", func(w http.ResponseWriter, r *http.Request) {
		api.PostFeedBack(db, w, r)
	}).Methods("POST")
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		api.LoginUser(db, w, r)
	}).Methods("POST")
	router.HandleFunc("/user/{userid}", func(w http.ResponseWriter, r *http.Request) {
		api.GetUserHttp(db, w, r)
	}).Methods("GET")
	router.HandleFunc("/user/{userid}", func(w http.ResponseWriter, r *http.Request) {
		api.UpdateUser(db, w, r)
	}).Methods("PUT")
	router.HandleFunc("/user/{userid}", func(w http.ResponseWriter, r *http.Request) {
		api.DeleteUser(db, w, r)
	}).Methods("DELETE")

	// File server for public images
	fs := http.FileServer(http.Dir(private_channel.GetPublicDir()))
	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", fs))

	// Add CORS support
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	go handleWsMessages()

	// Configure HTTPS server
	server := &http.Server{
		Addr:      ":8443", // HTTPS typically runs on port 443, but for development we use 8443
		Handler:   handler,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS10}, // Enforce TLS 1.3
	}

	// Provide paths to your certificate and key files
	certFile := "./service-v2.aiiyou.cn/fullchain.pem"
	keyFile := "./service-v2.aiiyou.cn/privkey.pem"

	log.Info("Server started on https://localhost:8443")
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Error("ListenAndServeTLS: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Infof("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	// Check if the user already has a WebSocket connection
	if existingConn, ok := clients[userID]; ok {
		log.Infof("User %s already has a WebSocket connection. Closing the old one.", userID)
		existingConn.WsConn.Close()
		delete(clients, userID)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", private_channel.UDPServerAddress)
	if err != nil {
		log.Errorf("Failed to resolve UDP address: %v", err)
		mutex.Unlock()
		return
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Errorf("Failed to dial UDP address: %v", err)
		mutex.Unlock()
		return
	}

	userIdU, _ := strconv.ParseUint(userID, 10, 64)
	pUdpConn := private_channel.NewPudpConn(userIdU, udpConn, nil, ws, private_channel.HandlePEvent)
	clients[userID] = pUdpConn
	mutex.Unlock()

	go handlePings(ws)
	defer func() {
		mutex.Lock()
		delete(clients, userID)
		mutex.Unlock()
		pUdpConn.UdpConnStop()
	}()

	go pUdpConn.ReadFromUDP()
	readFromWebSocket(ws, userID)
}

func readFromWebSocket(ws *websocket.Conn, userID string) {
	for {
		var msg private_channel.WsMessage
		if err := ws.ReadJSON(&msg); err != nil {
			log.Infof("WebSocket read error from user %s: %v", userID, err)
			break
		}
		log.Infof("Received message from user %s: %v", userID, msg)
		if _, ok := clients[userID]; !ok {
			log.Errorf("clients[] not exist, will close ws", userID)
			break
		}
		msg.UserID = userID
		if msg.BizInfo.JsonParams == nil {
			msg.BizInfo.JsonParams = make(map[string]interface{})
		}
		broadcast <- msg
	}
}

func handleWsMessages() {
	for msg := range broadcast {
		log.Infof("handleWsMessages:%v, clients:%v", msg, clients)
		mutex.Lock()
		for userID, pUdpConn := range clients {
			log.Infof("Handling message for user: %s, message:%v", userID, msg)
			msgUserIdUint, err := strconv.ParseUint(msg.UserID, 10, 64)
			if err != nil {
				log.Warnf("Cannot parse Uint, msg.UserID: %v", msg)
				continue
			}

			if clients[userID].StreamId == msgUserIdUint {
				bizInfo, err := json.Marshal(&msg.BizInfo)
				if err != nil {
					log.Warn(err)
					continue
				}
				if len(msg.BizInfo.Cmd) == 0 {
					log.Warnf("msg.BizInfo.Cmd is empty, ignoring the message, msg: %v", msg)
					continue
				}

				//copy jsonParams to bizInfo.jsonParams
				for k, v := range msg.JsonParams {
					msg.BizInfo.JsonParams[k] = v
				}
				switch msg.BizInfo.Cmd {
				case "ASR", "IMAGE":
					if len(msg.FileName) == 0 {
						log.Warnf("file_name is empty, ignore the msg: %v", msg)
						continue
					}
					log.Infof("bizInfo: %s", string(bizInfo))

					promptAudio, err := api.GetPromptFromDbByName(db, msg.JsonParams, api.BIZ_PROMPT_AUDIO)
					if err == nil && len(promptAudio) > 0 {
						log.Info("get prompt from db:", promptAudio)
						msg.BizInfo.JsonParams["promptAudio"] = promptAudio
					}

					promptImage, err := api.GetPromptFromDbByName(db, msg.JsonParams, api.BIZ_PROMPT_IMAGE)
					if err == nil && len(promptImage) > 0 {
						log.Info("get prompt from db:", promptImage)
						msg.BizInfo.JsonParams["promptImage"] = promptImage
					}

					bizInfo, _ = json.Marshal(&msg.BizInfo)
					log.Info("modified bizInfo: ", string(bizInfo))

					fileUri := filepath.Join(private_channel.GetUserUploadDir(msg.UserID), msg.FileName)
					fileContent, err := os.ReadFile(fileUri)
					if err != nil {
						log.Error(err)
						continue
					}
					if err := pUdpConn.SendPrivateMessage(&private_channel.PrivateMessage{
						StreamId:    msgUserIdUint,
						Tid:         uint64(msg.ReqId),
						BizInfo:     string(bizInfo),
						Content:     fileContent,
						IsReliable:  true,
						IsStreaming: false,
					}); err != nil {
						log.Warnf("UDP send error: %v", err)
						continue
					}

				default:
					//CHAT or Prompt
					finalContent := msg.Content
					if bizPrompt, exist := msg.JsonParams[api.BIZ_PROMPT]; exist {
						bizPromptStr, ok := bizPrompt.(string)
						if !ok {
							log.Error("Error: BIZ_PROMPT is not a string")
							continue
						}
						log.Info("bizPrompt:", bizPromptStr)
						bizFinalPromptStr:=bizPromptStr
						if !strings.Contains(bizPromptStr, "# ")  && len(bizPromptStr) < 64 {
							promtpDb, err := api.GetPrompt(db, bizPromptStr)
							if err != nil {
								log.Error(err)
								continue
							}
							bizFinalPromptStr=promtpDb.Prompt
						}

						finalContent = api.ReplacePlaceholders(bizFinalPromptStr, msg.JsonParams, msg.Content)
					}
					log.Info("finalContent:", finalContent)

					if err := pUdpConn.SendPrivateMessage(&private_channel.PrivateMessage{
						StreamId:    msgUserIdUint,
						BizInfo:     string(bizInfo),
						Content:     []byte(finalContent),
						IsReliable:  true,
						IsStreaming: false,
						Tid:         uint64(msg.ReqId),
					}); err != nil {
						log.Infof("UDP send error: %v", err)

					}
				}

			} else {
				log.Warnf("client[%s] streamId:%d, msgUserIdUint: %d, msg:%s will be ignore!", userID, clients[userID].StreamId, msgUserIdUint, msg)
			}
		}
		mutex.Unlock()
	}
}

func handlePings(ws *websocket.Conn) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			log.Infof("WebSocket ping error: %v", err)
			ws.Close()
			return
		}
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]
	log.Info("uploadFile incoming")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
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

	userUploadDir := private_channel.GetUserUploadDir(userID)
	if err := private_channel.EnsureDir(userUploadDir); err != nil {
		log.Infof("Directory creation error: %v", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(userUploadDir, handler.Filename))
	if err != nil {
		log.Infof("File creation error: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Infof("File copy error: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Successfully uploaded file: %s", handler.Filename)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userid"]
	filename := mux.Vars(r)["filename"]

	userDownloadDir := private_channel.GetUserDownloadDir(userID)
	filepath := filepath.Join(userDownloadDir, filename)
	download(w, filepath, filename)
}

func downloadPublicFile(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]

	publicDir := private_channel.GetPublicDir()
	filepath := filepath.Join(publicDir, filename)
	download(w, filepath, filename)
}

func download(w http.ResponseWriter, filepath, filename string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Errorf("File open error: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, file); err != nil {
		log.Infof("File copy error: %v", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
}
