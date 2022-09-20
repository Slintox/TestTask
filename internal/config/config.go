package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	PathToFiles string `yaml:"path_to_files"`
}

func NewConfig(configPath string) (*Config, error) {
	var config Config

	err := cleanenv.ReadConfig(configPath, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
