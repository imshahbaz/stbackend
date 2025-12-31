package model

type MongoEnvConfig struct {
	ID           string   `json:"-" bson:"_id,omitempty"`
	FrontendUrls []string `json:"frontendUrls" bson:"frontendUrls"`
	BrevoEmail   string   `json:"brevoEmail" bson:"brevoEmail"`
	BrevoApiKey  string   `json:"brevoApiKey" bson:"brevoApiKey"`
	ApiKey       string   `json:"apiKey" bson:"apiKey"`
	Leverage     float32  `json:"leverage" bson:"leverage"`
	DebugMode    bool     `json:"debugMode" bson:"debugMode"`
	RateLimiter  bool     `json:"rateLimiter" bson:"rateLimiter"`
	JwtSecret    string   `json:"jwtSecret" bson:"jwtSecret"`
}

type EnvConfig struct {
	Port          string `json:"port"`
	MongoUser     string `json:"mongoUser"`
	MongoPassword string `json:"mongoPassword"`
	Environment   string `json:"environment"`
}

// --- Huma Structs ---

type UpdateConfigInput struct {
	Body MongoEnvConfig
}

type ConfigResponse struct {
	Body MongoEnvConfig
}
