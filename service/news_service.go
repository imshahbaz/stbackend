package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type NewsService interface {
	FetchTVNews(symbol string) (*model.DefaultResponse, error)
}

type NewsServiceImpl struct {
	client *resty.Client
}

func NewNewsService() NewsService {
	c := resty.New()
	c.SetTimeout(15 * time.Second)

	// Set common headers for all requests in this service
	c.SetHeaders(map[string]string{
		"Referer":    "https://in.tradingview.com/",
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1",
	})

	return &NewsServiceImpl{
		client: c,
	}
}

func (svc *NewsServiceImpl) FetchTVNews(symbol string) (*model.DefaultResponse, error) {
	var result model.TVNewsResponse
	if ok, _ := database.RedisHelper.GetAsStruct("tv_news_"+symbol, result); ok {
		return &model.DefaultResponse{
			Body: model.Response{
				Success: true,
				Data:    result.Items,
			},
		}, nil
	}

	resp, err := svc.client.R().
		SetResult(&result).
		SetQueryParams(map[string]string{
			"client":         "chart",
			"user_prostatus": "non_pro",
		}).
		SetQueryParam("filter", "lang:en").
		SetQueryParam("filter", "symbol:NSE:"+symbol).
		Get("https://news-mediator.tradingview.com/public/news-flow/v2/news")

	if err != nil || resp.StatusCode() != http.StatusOK {
		log.Err(err).Msgf("Error calling tv api %v", resp)
		return &model.DefaultResponse{
			Body: model.Response{
				Success: false,
				Data:    []string{},
			},
		}, nil
	}

	cache.GoSet("tv_news_"+symbol, result, 10*time.Minute)
	return &model.DefaultResponse{
		Body: model.Response{
			Success: true,
			Data:    result.Items,
		},
	}, nil
}
