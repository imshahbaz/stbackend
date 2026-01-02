package model

type MongoEnvConfig struct {
	ID           string                `json:"-" bson:"_id,omitempty"`
	FrontendUrls []string              `json:"frontendUrls" bson:"frontendUrls"`
	BrevoEmail   string                `json:"brevoEmail" bson:"brevoEmail"`
	BrevoApiKey  string                `json:"brevoApiKey" bson:"brevoApiKey"`
	ApiKey       string                `json:"apiKey" bson:"apiKey"`
	Leverage     float32               `json:"leverage" bson:"leverage"`
	DebugMode    bool                  `json:"debugMode" bson:"debugMode"`
	RateLimiter  bool                  `json:"rateLimiter" bson:"rateLimiter"`
	JwtSecret    string                `json:"jwtSecret" bson:"jwtSecret"`
	RedisUrl     string                `json:"redisUrl" bson:"redisUrl"`
	RedisCache   bool                  `json:"redisCache" bson:"redisCache"`
	GoogleAuth   GoogleAuthCredentials `json:"googleAuth" bson:"googleAuth"`
}

type EnvConfig struct {
	Port          string `json:"port"`
	MongoUser     string `json:"mongoUser"`
	MongoPassword string `json:"mongoPassword"`
	Environment   string `json:"environment"`
}

type UpdateConfigInput struct {
	Body MongoEnvConfig
}

type ConfigResponse struct {
	Body MongoEnvConfig
}

type GoogleAuthCredentials struct {
	ClientID     string `json:"clientId" bson:"clientId"`
	ClientSecret string `json:"secret" bson:"secret"`
	CallbackUrl  string `json:"callbackUrl" bson:"callbackUrl"`
}

type ClientConfigs struct {
	ID   string `json:"-" bson:"_id,omitempty"`
	Auth struct {
		Google     bool `json:"google" bson:"google"`
		Email      bool `json:"email" bson:"email"`
		TrueCaller bool `json:"truecaller" bson:"truecaller"`
	} `json:"auth" bson:"auth"`
}
