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
	KafkaBrokers           []string `mapstructure:"KAFKA_BROKERS"`
	KafkaGroupID           string   `mapstructure:"KAFKA_GROUP_ID"`
	KafkaTopicFileUploaded string   `mapstructure:"KAFKA_TOPIC_FILE_UPLOADED"`
	KafkaTopicProcessed    string   `mapstructure:"KAFKA_TOPIC_FILE_PROCESSED"`
}

func NewEnv() *Env {
	env := Env{}
	viper.SetConfigFile(".env")

	// 2. Cho phép tự động đọc biến môi trường hệ thống (Docker / K8s / OS env)
	viper.AutomaticEnv()
	// 3. Đặt giá trị mặc định phòng trường hợp thiếu biến
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("SERVER_ADDRESS", ":8080")
	viper.SetDefault("CONTEXT_TIMEOUT", 10)
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("MINIO_BUCKET", "file-uploads")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("KAFKA_GROUP_ID", "file-upload-service")
	viper.SetDefault("KAFKA_TOPIC_FILE_UPLOADED", "file.uploaded")
	viper.SetDefault("KAFKA_TOPIC_FILE_PROCESSED", "file.processed")

	// 4. Đọc file .env nếu có (không crash nếu không có file, để deploy Docker/K8s)
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("[INFO] No .env file found, using system environment variables: %v", err)
	} else {
		log.Printf("[INFO] Loaded configuration from .env file: %s", viper.ConfigFileUsed())
	}
	// 5. Unmarshal vào struct
	if err := viper.Unmarshal(&env); err != nil {
		log.Fatalf("Failed to unmarshal env configuration: %v", err)
	}
	if env.AppEnv == "development" {
		log.Println("[INFO] Application is running in DEVELOPMENT mode")
	}
	return &env

}
