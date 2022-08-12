package config

import "github.com/spf13/viper"

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.BindEnv("DIR")
	viper.SetDefault("dir", "/var/gemfast")
	viper.AutomaticEnv()
}

func Get(k string) (interface {}) {
	return viper.Get(k)
}