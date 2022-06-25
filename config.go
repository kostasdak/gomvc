package gomvc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	UseCache         bool
	Server           ServerConf
	Database         DatabaseConf
	EnableInfoLog    bool
	ShowStackOnError bool
}

type ServerConf struct {
	Port          int
	SessionSecure bool
}

type DatabaseConf struct {
	Server string
	Dbname string
	Dbuser string
	Dbpass string
}

type configValues map[string]interface{}

var ncfg configValues

func ReadConfig(filePath string) *AppConfig {
	ncfg = make(configValues)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	lastSection := ""
	for scanner.Scan() {
		line := scanner.Text()
		tline := strings.Trim(line, " ")

		if !strings.HasPrefix(tline, "#") {
			var nvPair []string

			// subArray
			if strings.HasSuffix(tline, ":") {
				if tline == "/:" {
					lastSection = ""
				} else {
					lastSection = tline
				}
			} else {
				// Value
				i := strings.Index(tline, ":")
				if i > 0 {
					nvPair = getValuePair(tline, ":")
				} else {
					i = strings.Index(tline, "=")
					if i > 0 {
						nvPair = getValuePair(tline, "=")
					}
				}

				if len(nvPair) == 2 {
					// string, bool, number
					i, err := strconv.ParseInt(nvPair[1], 10, 64)
					if err == nil {
						ncfg.Add(lastSection+nvPair[0], int(i))
						continue
					}
					f, err := strconv.ParseFloat(nvPair[1], 64)
					if err == nil {
						ncfg.Add(lastSection+nvPair[0], f)
						continue
					}
					b, err := strconv.ParseBool(nvPair[1])
					if err == nil {
						ncfg.Add(lastSection+nvPair[0], b)
						continue
					}
					if strings.HasPrefix(nvPair[1], "\"") && strings.HasSuffix(nvPair[1], "\"") {
						ncfg.Add(lastSection+strings.Trim(nvPair[0], "\""), strings.Trim(nvPair[1], "\""))
						continue
					}
					ncfg.Add(lastSection+nvPair[0], nvPair[1])
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
	// Unmarshal
	conf := unmarshal(ncfg)
	return conf
}

func (*AppConfig) GetValue(key string) interface{} {
	return ncfg.Get(key)
}

func unmarshal(ncfg configValues) *AppConfig {
	conf := &AppConfig{}

	if ncfg.Get("UseCache") != nil {
		conf.UseCache = ncfg.Get("UseCache").(bool)
	}
	if ncfg.Get("EnableInfoLog") != nil {
		conf.EnableInfoLog = ncfg.Get("EnableInfoLog").(bool)
	}
	if ncfg.Get("ShowStackOnError") != nil {
		conf.ShowStackOnError = ncfg.Get("ShowStackOnError").(bool)
	}
	if ncfg.Get("server:port") != nil {
		conf.Server.Port = ncfg.Get("server:port").(int)
	}
	if ncfg.Get("server:SessionSecure") != nil {
		conf.Server.SessionSecure = ncfg.Get("server:SessionSecure").(bool)
	}

	conf.Database.Server = fmt.Sprint(ncfg.Get("database:server"))
	conf.Database.Dbname = fmt.Sprint(ncfg.Get("database:dbname"))
	conf.Database.Dbuser = fmt.Sprint(ncfg.Get("database:dbuser"))
	conf.Database.Dbpass = fmt.Sprint(ncfg.Get("database:dbpass"))

	return conf
}

func getValuePair(s string, sep string) []string {
	nvPair := strings.Split(s, sep)
	return []string{strings.Trim(nvPair[0], " "), strings.Trim(nvPair[1], " ")}
}

func (s *configValues) Add(k string, v interface{}) {
	r := *s
	r[k] = v
}

func (s *configValues) Get(k string) interface{} {
	r := *s
	return r[k]
}
