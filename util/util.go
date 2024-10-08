package util

import (
	"github.com/spf13/viper"
)

func LoadConfig() {
	viper.SetConfigName("local")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
