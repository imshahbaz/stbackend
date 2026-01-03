package database

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
	"github.com/valkey-io/valkey-go"
)

var (
	RedisHelper *valkeyUtil
)

type valkeyUtil struct {
	client valkey.Client
}

func InitRedis(uri string) {
	opts, err := valkey.ParseURL(uri)
	if err != nil {
		log.Fatal().Msgf("Invalid Valkey URI: %v", err)
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       opts.InitAddress,
		TLSConfig:         opts.TLSConfig,
		BlockingPoolSize:  100,
		SelectDB:          opts.SelectDB,
		PipelineMultiplex: 2,
		Username:          opts.Username,
		Password:          opts.Password,
	})

	if err != nil {
		log.Fatal().Msgf("Could not connect to Valkey: %v", err)
	}

	err = client.Do(context.Background(), client.B().Ping().Build()).Error()
	if err != nil {
		log.Fatal().Msgf("Valkey Ping Failed: %v", err)
	}

	log.Info().Msg("✅ Connected to Aiven Valkey successfully")

	RedisHelper = &valkeyUtil{
		client: client,
	}
}

func (v *valkeyUtil) Set(key string, value any, expiration time.Duration) error {
	var data string

	switch val := value.(type) {
	case string:
		data = val
	case []byte:
		data = string(val)
	default:
		marshaled, err := sonic.ConfigDefault.MarshalToString(value)
		if err != nil {
			return fmt.Errorf("sonic marshal error: %w", err)
		}
		data = marshaled
	}

	cmd := v.client.B().Set().Key(key).Value(data).Ex(expiration).Build()
	return v.client.Do(context.Background(), cmd).Error()
}

func (v *valkeyUtil) GetAsStruct(key string, target any) (bool, error) {
	cmd := v.client.B().Get().Key(key).Build()
	resp := v.client.Do(context.Background(), cmd)

	if valkey.IsValkeyNil(resp.Error()) {
		return false, nil
	}

	if resp.Error() != nil {
		return false, fmt.Errorf("valkey execution error: %w", resp.Error())
	}

	val, err := resp.ToString()
	if err != nil {
		return false, fmt.Errorf("valkey get error: %w", err)
	}

	if strPtr, ok := target.(*string); ok {
		*strPtr = val
		return true, nil
	}

	err = sonic.ConfigDefault.UnmarshalFromString(val, target)
	if err != nil {
		return false, fmt.Errorf("sonic unmarshal error: %w", err)
	}

	return true, nil
}

func (v *valkeyUtil) Delete(key string) error {
	cmd := v.client.B().Del().Key(key).Build()
	return v.client.Do(context.Background(), cmd).Error()
}

func (v *valkeyUtil) Exists(key string) bool {
	cmd := v.client.B().Exists().Key(key).Build()
	count, err := v.client.Do(context.Background(), cmd).AsInt64()
	return err == nil && count > 0
}
