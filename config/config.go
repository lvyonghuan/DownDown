package config

import "github.com/spf13/viper"

// ReadConfig 读取配置文件
func ReadConfig() (Config, error) {
	config := new(Config)

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}
	if err := viper.Unmarshal(config); err != nil {
		return Config{}, err
	}

	return *config, nil
}
