package service

import (
	"encoding/binary"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type AngelOneWebSocket struct {
	Conn          *websocket.Conn
	mu            sync.RWMutex
	stockChannels map[string]chan float64
}

func NewAngelOneWebSocket(jwt, apiKey, clientCode, feedToken string) *AngelOneWebSocket {
	header := http.Header{}
	header.Add("Authorization", "Bearer "+jwt)
	header.Add("x-api-key", apiKey)
	header.Add("x-client-code", clientCode)
	header.Add("x-feed-token", feedToken)

	url := "wss://smartapisocket.angelone.in/smart-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Smart Stream")
		return nil
	}

	aws := &AngelOneWebSocket{
		Conn:          conn,
		stockChannels: make(map[string]chan float64),
	}

	go aws.readLoop()
	go aws.heartbeatLoop()

	return aws
}

func (aws *AngelOneWebSocket) readLoop() {
	defer aws.Disconnect()

	for {
		_, message, err := aws.Conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Read error in AngelOneWebSocket")
			return
		}

		if len(message) == 4 && string(message) == "pong" {
			log.Info().Msg("Pong received")
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

func (aws *AngelOneWebSocket) Subscribe(token string, ch chan float64) {
	aws.mu.Lock()
	aws.stockChannels[token] = ch
	aws.mu.Unlock()

	// Fixed Structure: mode and tokenList must be inside "params"
	request := map[string]any{
		"correlationId": "shahbaz_trail",
		"action":        1,
		"params": map[string]any{
			"mode": 1, // 1 = LTP
			"tokenList": []map[string]any{
				{
					"exchangeType": 1, // 1 = NSE
					"tokens":       []string{token},
				},
			},
		},
	}

	data, err := sonic.Marshal(request)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal subscription")
		return
	}

	aws.mu.Lock()
	err = aws.Conn.WriteMessage(websocket.TextMessage, data)
	aws.mu.Unlock()

	if err != nil {
		log.Error().Err(err).Str("token", token).Msg("Failed to send subscription")
	} else {
		log.Info().Str("token", token).Msg("📡 Subscribed (Server request sent)")
	}
}

func (aws *AngelOneWebSocket) Unsubscribe(token string) {
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
	data, _ := sonic.Marshal(request)
	aws.mu.Lock()
	defer aws.mu.Unlock()
	aws.Conn.WriteMessage(websocket.TextMessage, data)
}

func (aws *AngelOneWebSocket) heartbeatLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		aws.mu.Lock()
		if aws.Conn == nil {
			aws.mu.Unlock()
			return
		}
		err := aws.Conn.WriteMessage(websocket.TextMessage, []byte("ping"))
		aws.mu.Unlock()

		if err != nil {
			return
		}
	}
}

func (aws *AngelOneWebSocket) Disconnect() {
	aws.mu.Lock()
	defer aws.mu.Unlock()
	if aws.Conn != nil {
		aws.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		aws.Conn.Close()
		aws.Conn = nil
	}
}
