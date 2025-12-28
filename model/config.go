package model

var (
	ObCollectionName = "OrderBlocks"
)

type MongoEnvConfig struct {
	ID          string  `json:"-" bson:"_id,omitempty"`
	FrontendUrl string  `json:"frontendUrl" bson:"frontendUrl"`
	BrevoEmail  string  `json:"brevoEmail" bson:"brevoEmail"`
	BrevoApiKey string  `json:"brevoApiKey" bson:"brevoApiKey"`
	ApiKey      string  `json:"apiKey" bson:"apiKey"`
	Leverage    float32 `json:"leverage" bson:"leverage"`
	DebugMode   bool    `json:"debug" bson:"debug"`
	RateLimiter bool    `json:"rateLimiter" bson:"rateLimiter"`
	JwtSecret   string  `json:"jwtSecret" bson:"jwtSecret"`
}

// --- SYSTEM CONFIG ---
// EnvConfig holds sensitive environment settings
// @Description Private configuration (usually not exposed in public endpoints)
type EnvConfig struct {
	Port          string `json:"port"`
	MongoUser     string `json:"mongoUser"`
	MongoPassword string `json:"mongoPassword"`
	Environment   string `json:"environment"`
}
