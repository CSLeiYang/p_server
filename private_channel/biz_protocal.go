package private_channel

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "csleiyang.com/p_server/logger"
)

type PCommand struct {
	SID        string                 `json:"sid"`
	Cmd        string                 `json:"cmd"`
	Params     string                 `json:"params"`
	JsonParams map[string]interface{} `json:"json_params"`
}

type PEvent struct {
	SID          string `json:"sid"`
	EventType    string `json:"event_type"`
	EventContent string `json:"event_content"`
	EventParams  string `json:"event_params"`
}

type WsMessage struct {
	UserID     string                 `json:"userid"`
	ReqId      uint32                 `json:"reqid"`
	BizInfo    PCommand               `json:"biz_info"`
	Content    string                 `json:"content"`
	FileName   string                 `json:"file_name"`
	JsonParams map[string]interface{} `json:"json_params"`
}

type WsEvent struct {
	UserID   string `json:"userid"`
	ReqId    uint32 `json:"reqid"`
	RespId   uint32 `json:"respid"`
	Event    PEvent `json:"event"`
	Content  string `json:"content"`
	FileName string `json:"file_name"`
}

func HandlePEvent(userId uint64, isStream, wholePM bool, bizInfo string, content []byte, pUdpCon *PUdpConn) error {
	log.Info("bizInfo: ", bizInfo)
	if len(bizInfo) == 0 {
		log.Error("bizInfo is empty")
		return errors.New("bizInfo is empty")
	}

	if pUdpCon == nil || pUdpCon.WsConn == nil {
		log.Error("pUdpCon or pUdpCon.WsConn is nil")
		return errors.New("pUdpCon or pUdpCon.WsConn is nil")
	}

	userIdStr := strconv.FormatUint(userId, 10)

	if isStream {
		bizinfoUint, _ := strconv.ParseUint(bizInfo, 10, 32)
		pUdpCon.WsConn.WriteJSON(&WsEvent{
			UserID:  userIdStr,
			RespId:  uint32(bizinfoUint),
			Content: string(content),
		})

	} else {
		if strings.Contains(bizInfo, "Heartbeat") {
			log.Info("Hearbeat from server")
			return nil
		}
		var pEvent PEvent
		err := json.Unmarshal([]byte(bizInfo), &pEvent)
		if err != nil {
			log.Error(err)
			return err
		}

		eventParamsSplits := strings.Split(pEvent.EventParams, "|")
		reqIdStr := eventParamsSplits[0]
		reqId, _ := strconv.ParseUint(reqIdStr, 10, 32)
		switch pEvent.EventType {
		case "TTS", "TTS_STREAM":
			fileName := strings.ReplaceAll(pEvent.EventParams, "|", "_")
			userDownloadDir := GetUserDownloadDir(userIdStr)
			if err := EnsureDir(userDownloadDir); err != nil {
				log.Errorf("Directory creation error: %v", err)
				return err
			}
			err := os.WriteFile(filepath.Join(userDownloadDir, fileName), content, 0644)
			if err != nil {
				log.Error(err)
				return err
			}
			pUdpCon.WsConn.WriteJSON(&WsEvent{
				UserID:   userIdStr,
				Event:    pEvent,
				ReqId:    uint32(reqId),
				RespId:   uint32(reqId),
				FileName: fileName,
			})
		case "IMAGE":
			fileName := ""
			if len(eventParamsSplits) > 1 {
				fileName = eventParamsSplits[1]
			}
			pUdpCon.WsConn.WriteJSON(&WsEvent{
				UserID:   userIdStr,
				Event:    pEvent,
				ReqId:    uint32(reqId),
				RespId:   uint32(reqId),
				Content:  string(content),
				FileName: fileName,
			})

		default:
			log.Info("content:", string(content))
			pUdpCon.WsConn.WriteJSON(&WsEvent{
				UserID:  userIdStr,
				Event:   pEvent,
				ReqId:   uint32(reqId),
				RespId:  uint32(reqId),
				Content: string(content),
			})

		}
	}

	return nil
}
