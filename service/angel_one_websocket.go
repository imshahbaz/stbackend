package service

import (
	"backend/model"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var ErrConnectionClosed = errors.New("websocket connection is nil or closed")

type AngelOneWebSocket interface {
	Subscribe(token string) (chan float64, error)
	Unsubscribe(token string)
	Disconnect()
	StartWebsocket() error
	UpdateConfig(jwt, feedToken string)
	StopUpdateChannel()
}

type AngelOneWebSocketImpl struct {
	conn          *websocket.Conn
	mu            sync.RWMutex
	writeMu       sync.Mutex
	stockChannels map[string]chan float64
	jwt           string
	apiKey        string
	clientCode    string
	feedToken     string
	config        *model.AngelOneConfig
}

func NewAngelOneWebSocket(jwt, feedToken string, config *model.AngelOneConfig) AngelOneWebSocket {
	return &AngelOneWebSocketImpl{
		stockChannels: make(map[string]chan float64),
		jwt:           jwt,
		apiKey:        config.ApiKey,
		clientCode:    config.ClientID,
		feedToken:     feedToken,
		config:        config,
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

			aws.mu.RLock()
			ch, exists := aws.stockChannels[token]
			aws.mu.RUnlock()

			if exists {
				select {
				case ch <- ltp:
				default:
				}
			}
		}
	}
}

func (aws *AngelOneWebSocketImpl) Subscribe(token string) (chan float64, error) {
	aws.mu.Lock()
	ch, exists := aws.stockChannels[token]
	if exists {
		aws.mu.Unlock()
		return ch, nil
	}

	ch = make(chan float64, 100)
	aws.stockChannels[token] = ch
	aws.mu.Unlock()

	request := map[string]any{
		"correlationId": "shahbaz_trades",
		"action":        1,
		"params": map[string]any{
			"mode": 1,
			"tokenList": []map[string]any{
				{
					"exchangeType": 1,
					"tokens":       []string{token},
				},
			},
		},
	}

	return ch, aws.safeWrite(websocket.TextMessage, request)
}

func (aws *AngelOneWebSocketImpl) Unsubscribe(token string) {
	aws.mu.Lock()
	if ch, exists := aws.stockChannels[token]; exists {
		close(ch)
		delete(aws.stockChannels, token)
	}
	aws.mu.Unlock()

	request := map[string]any{
		"correlationId": "shahbaz_trail",
		"action":        2,
		"mode":          1,
		"tokenList": []map[string]any{
			{"exchangeType": 1, "tokens": []string{token}},
		},
	}

	aws.safeWrite(websocket.TextMessage, request)
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
	aws.mu.Lock()
	if aws.conn != nil {
		aws.safeWrite(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		aws.conn.Close()
		aws.conn = nil
	}
	aws.mu.Unlock()

	aws.mu.Lock()
	for token, ch := range aws.stockChannels {
		close(ch)
		delete(aws.stockChannels, token)
	}
	aws.mu.Unlock()
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
