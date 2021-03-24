package configs

import (
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("waf")
	viper.AutomaticEnv()
	viper.SetDefault("image", "arthurcgc/waf-modsecurity")
	viper.SetDefault("port", "8080")
	viper.SetDefault("outside_cluster", "false")
}
