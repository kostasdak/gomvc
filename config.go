package gomvc

import (
	"fmt"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Title            string // this is the app title
	UseCache         bool
	Server           ServerConf   `mapstructure:"server"`
	Database         DatabaseConf `mapstructure:"database"`
	EnableInfoLog    bool
	InfoFile         string
	ShowStackOnError bool
	ErrorFile        string
}

type ServerConf struct {
	Port           int  // Server port
	SessionTimeout int  // ???
	SessionSecure  bool //http / https
}

type DatabaseConf struct {
	ConnectionString string
	Server           string
	Dbname           string
	Dbuser           string
	Dbpass           string
}

func LoadConfig(filePath string) *AppConfig {

	conf := &AppConfig{}

	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Read Config Error")
		fmt.Println(err)
	}

	err = viper.Unmarshal(conf)
	if err != nil {
		fmt.Println(err)
	}

	return conf
}
