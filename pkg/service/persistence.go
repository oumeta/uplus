package service

type PersistenceService interface {
	NewStore(id string, subIDs ...string) Store
}

type Store interface {
	Load(val interface{}) error
	Save(val interface{}) error
	Reset() error
}

type RedisPersistenceConfig struct {
	Host     string `yaml:"host" json:"host" env:"REDIS_HOST"`
	Port     string `yaml:"port" json:"port" env:"REDIS_PORT"`
	Password string `yaml:"password,omitempty" json:"password,omitempty" env:"REDIS_PASSWORD"`
	DB       int    `yaml:"db" json:"db" env:"REDIS_DB"`
}

type JsonPersistenceConfig struct {
	Directory string `yaml:"directory" json:"directory"`
}
