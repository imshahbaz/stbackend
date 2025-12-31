package model

type TruecallerDto struct {
	AccessToken string `json:"accessToken"`
	RequestId   string `json:"requestId"`
	Endpoint    string `json:"endpoint"`
}

type TruecallerProfile struct {
	ID     string `json:"id"`
	UserID int64  `json:"userId"`
	Name   struct {
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"name"`
	PhoneNumbers     []int64 `json:"phoneNumbers"`
	OnlineIdentities struct {
		Email string `json:"email"`
	} `json:"onlineIdentities"`
	Addresses []struct {
		CountryCode string `json:"countryCode"`
	} `json:"addresses"`
	Gender   string         `json:"gender"`
	IsActive bool           `json:"isActive"`
	Privacy  string         `json:"privacy"`
	Type     string         `json:"type"`
	Badges   []any          `json:"badges"`
	History  map[string]any `json:"history"`
}
