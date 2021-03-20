package configs

import (
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("waf")
	viper.AutomaticEnv()
	viper.SetDefault("image", "arthurcgc/waf-modsecurity:v1")
	viper.SetDefault("port", "8080")
	viper.SetDefault("outside_cluster", "false")
	viper.AddConfigPath(".")
	viper.SetConfigName("modsecurity-recommended")
	viper.ReadInConfig()
}
