package service

import (
	"backend/model"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var ErrConnectionClosed = errors.New("websocket connection is nil or closed")

type AngelOneWebSocket interface {
	Subscribe(token string, exchangeType model.ExchangeType) error
	Unsubscribe(token string, exchangeType model.ExchangeType)
	Disconnect()
	StartWebsocket() error
	UpdateConfig(jwt, feedToken string)
	StopUpdateChannel()
	GetLTP(token string) float64
}

type AngelOneWebSocketImpl struct {
	conn       *websocket.Conn
	mu         sync.RWMutex
	writeMu    sync.Mutex
	ltpCache   sync.Map
	jwt        string
	apiKey     string
	clientCode string
	feedToken  string
	config     *model.AngelOneConfig
	connected  atomic.Bool
}

func NewAngelOneWebSocket(jwt, feedToken string, config *model.AngelOneConfig) AngelOneWebSocket {
	return &AngelOneWebSocketImpl{
		jwt:        jwt,
		apiKey:     config.ApiKey,
		clientCode: config.ClientID,
		feedToken:  feedToken,
		config:     config,
	}
}

func (aws *AngelOneWebSocketImpl) readLoop() {
	defer aws.Disconnect()

	for {
		_, message, err := aws.conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Read error in AngelOneWebSocket")
			return
		}

		if len(message) == 4 && string(message) == "pong" {
			continue
		}

		if len(message) >= 51 && message[0] == 1 {
			token := strings.TrimRight(string(message[2:27]), "\x00")
			priceInt := binary.LittleEndian.Uint32(message[43:47])
			ltp := float64(priceInt) / 100.0
			aws.ltpCache.Store(token, ltp)
		}
	}
}

func (aws *AngelOneWebSocketImpl) Subscribe(token string, exchangeType model.ExchangeType) error {
	if _, ok := aws.ltpCache.Load(token); ok {
		return nil
	}

	aws.ltpCache.Store(token, -1.0)

	request := map[string]any{
		"correlationId": "shahbaz_trades",
		"action":        1,
		"params": map[string]any{
			"mode": 1,
			"tokenList": []map[string]any{
				{
					"exchangeType": exchangeType,
					"tokens":       []string{token},
				},
			},
		},
	}

	return aws.safeWrite(websocket.TextMessage, request)
}

func (aws *AngelOneWebSocketImpl) Unsubscribe(token string, exchangeType model.ExchangeType) {
	request := map[string]any{
		"correlationId": "shahbaz_trades",
		"action":        2,
		"mode":          1,
		"tokenList": []map[string]any{
			{"exchangeType": exchangeType, "tokens": []string{token}},
		},
	}

	aws.safeWrite(websocket.TextMessage, request)
	aws.ltpCache.Delete(token)
}

func (aws *AngelOneWebSocketImpl) heartbeatLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		aws.mu.Lock()
		if aws.conn == nil {
			aws.mu.Unlock()
			return
		}

		err := aws.safeWrite(websocket.TextMessage, []byte("ping"))
		aws.mu.Unlock()

		if err != nil {
			return
		}
	}
}

func (aws *AngelOneWebSocketImpl) Disconnect() {
	aws.connected.Store(false)
	aws.mu.Lock()
	if aws.conn != nil {
		aws.safeWrite(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		aws.conn.Close()
		aws.conn = nil
	}
	aws.mu.Unlock()

	aws.ltpCache = sync.Map{}
}

func (aws *AngelOneWebSocketImpl) StartWebsocket() error {
	aws.mu.Lock()
	defer aws.mu.Unlock()
	if aws.conn != nil {
		return nil
	}

	header := http.Header{}
	header.Add("Authorization", "Bearer "+aws.jwt)
	header.Add("x-api-key", aws.apiKey)
	header.Add("x-client-code", aws.clientCode)
	header.Add("x-feed-token", aws.feedToken)

	url := "wss://smartapisocket.angelone.in/smart-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Smart Stream")
		return err
	}

	aws.conn = conn

	go aws.readLoop()
	go aws.heartbeatLoop()

	aws.connected.Store(true)
	return nil
}

func (aws *AngelOneWebSocketImpl) safeWrite(msgType int, data any) error {
	var payload []byte
	var err error

	if b, ok := data.([]byte); ok {
		payload = b
	} else {
		payload, err = sonic.Marshal(data)
		if err != nil {
			return err
		}
	}

	aws.writeMu.Lock()
	defer aws.writeMu.Unlock()

	if aws.conn == nil {
		return ErrConnectionClosed
	}

	return aws.conn.WriteMessage(msgType, payload)
}

func (aws *AngelOneWebSocketImpl) UpdateConfig(jwt, feedToken string) {
	aws.mu.Lock()
	defer aws.mu.Unlock()
	aws.jwt = jwt
	aws.feedToken = feedToken
}

func (aws *AngelOneWebSocketImpl) StopUpdateChannel() {
	aws.mu.Lock()
	defer aws.mu.Unlock()
	if updateChan != nil {
		close(updateChan)
	}
}

func (aws *AngelOneWebSocketImpl) GetLTP(token string) float64 {
	if !aws.connected.Load() {
		return -2
	}

	val, ok := aws.ltpCache.Load(token)
	if !ok {
		return -1
	}
	return val.(float64)
}
