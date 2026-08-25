package bootstrap

import (
	"log"

	"github.com/spf13/viper"
)

type Env struct {
	AppEnv                 string `mapstructure:"APP_ENV"`
	ServerAddress          string `mapstructure:"SERVER_ADDRESS"`
	ContextTimeout         int    `mapstructure:"CONTEXT_TIMEOUT"`
	DBDriver               string `mapstructure:"DB_DRIVER"`
	DBHost                 string `mapstructure:"DB_HOST"`
	DBPort                 string `mapstructure:"DB_PORT"`
	DBUser                 string `mapstructure:"DB_USER"`
	DBPass                 string `mapstructure:"DB_PASS"`
	DBName                 string `mapstructure:"DB_NAME"`
	AccessTokenExpiryHour  int    `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour int    `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenSecret      string `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret     string `mapstructure:"REFRESH_TOKEN_SECRET"`

	// MinIO
	MinioEndpoint string `mapstructure:"MINIO_ENDPOINT"`
	MinioUser     string `mapstructure:"MINIO_ROOT_USER"`
	MinioPassword string `mapstructure:"MINIO_ROOT_PASSWORD"`
	MinioUseSSL   bool   `mapstructure:"MINIO_USE_SSL"`
	MinioBucket   string `mapstructure:"MINIO_BUCKET"`

	// Kafka
	KafkaBrokers []string `mapstructure:"KAFKA_BROKERS"`
}

func NewEnv() *Env {
	env := Env{}
	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Can not find .env file: ", err)
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("Can not unmarshal .env file: ", err)
	}

	if env.AppEnv == "development" {
		log.Println("Running in development mode")
	}
	return &env

}
