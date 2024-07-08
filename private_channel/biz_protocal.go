package private_channel

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	log "csleiyang.com/p_server/logger"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type PCommand struct {
	SID    string `json:"sid"`
	Cmd    string `json:"cmd"`
	Params string `json:"params"`
}

type PEvent struct {
	SID          string `json:"sid"`
	EventType    string `json:"event_type"`
	EventContent string `json:"event_content"`
}

func HandlePEvent(userId uint64, wholePM bool, bizInfo string, content []byte, pUdpCon *PUdpConn) error {
	log.Infof("HandlePEvent, bizInfo: %s, content:%v", bizInfo, string(content))
	if len(bizInfo) == 0 {
		log.Error("bizInfo is empty")
		return errors.New("bizInfo is empty")
	}

	var pEvent PEvent
	json.Unmarshal([]byte(bizInfo), &pEvent)
	switch pEvent.EventType {
	case "TTS":
		log.Info("TTS Text: ", pEvent.EventContent)
		// 将[]byte转换为io.Reader
		go func() {
			mp3Reader := bytes.NewReader(content)

			// 将io.Reader转换为ReadCloser
			mp3ReadCloser := io.NopCloser(mp3Reader)
			streamer, format, err := mp3.Decode(mp3ReadCloser)
			if err != nil {
				log.Error(err)
				return
			}
			defer streamer.Close()

			err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			if err != nil {
				log.Error(err)
				return
			}

			done := make(chan bool)
			speaker.Play(beep.Seq(streamer, beep.Callback(func() {
				done <- true
			})))
			time.Sleep(time.Millisecond * 100)
			streamer.Close()
			<-done

		}()

	case "CHAT":
		log.Info("AI: ", string(pEvent.EventContent))
	}
	return nil

}
