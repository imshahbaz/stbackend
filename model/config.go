package model

type MongoEnvConfig struct {
	DebugMode     bool     `bson:"debugMode" json:"debugMode"`
	Environment   string   `bson:"environment" json:"environment"`
	Port          string   `bson:"port" json:"port"`
	MongoUri      string   `bson:"mongoUri" json:"mongoUri"`
	JwtSecret     string   `bson:"jwtSecret" json:"jwtSecret"`
	BrevoApiKey   string   `bson:"brevoApiKey" json:"brevoApiKey"`
	BrevoEmail    string   `bson:"brevoEmail" json:"brevoEmail"`
	FrontendUrls  []string `bson:"frontendUrls" json:"frontendUrls"`
	RateLimiter   bool     `bson:"rateLimiter" json:"rateLimiter"`
	Leverage      float32  `bson:"leverage" json:"leverage"`
	MongoUser     string   `bson:"mongoUser" json:"mongoUser"`
	MongoPassword string   `bson:"mongoPassword" json:"mongoPassword"`
}

type EnvConfig = MongoEnvConfig

// --- Huma Structs ---

type UpdateConfigInput struct {
	Body MongoEnvConfig
}

type ConfigResponse struct {
	Body MongoEnvConfig
}
