package gomvc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AppConfig is the application config,
type AppConfig struct {
	UseCache         bool
	Server           ServerConf
	Database         DatabaseConf
	EnableInfoLog    bool
	ShowStackOnError bool
	RateLimit        RateLimitConf
}

// ServerConf http listening port and true/false option for https
type ServerConf struct {
	Port          int
	SessionSecure bool
}

// DatabaseConf set MySql server address, database name, username and password
type DatabaseConf struct {
	Server string
	Port   int // Add this
	Dbname string
	Dbuser string
	Dbpass string
	UseTLS bool // Add this
}

// RateLimitConf for rate limiting configuration
type RateLimitConf struct {
	Enabled              bool
	IPMaxAttempts        int
	IPBlockMinutes       int
	UsernameMaxAttempts  int
	UsernameBlockMinutes int
}

// configValues is the map that holds the configuration values
type configValues map[string]interface{}

var ncfg configValues

// ReadConfig this function is for reading the configuration file
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

// GetValue get a parammeter value from a specific key
func (*AppConfig) GetValue(key string) interface{} {
	return ncfg.Get(key)
}

// unmarshal internal function to apply the file parameters to gomvc variables
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
	if ncfg.Get("server:Port") != nil {
		conf.Server.Port = ncfg.Get("server:port").(int)
	}
	if ncfg.Get("server:SessionSecure") != nil {
		conf.Server.SessionSecure = ncfg.Get("server:SessionSecure").(bool)
	}

	conf.Database.Server = fmt.Sprint(ncfg.Get("database:server"))
	conf.Database.Dbname = fmt.Sprint(ncfg.Get("database:dbname"))
	conf.Database.Dbuser = fmt.Sprint(ncfg.Get("database:dbuser"))
	conf.Database.Dbpass = fmt.Sprint(ncfg.Get("database:dbpass"))

	// Database port with default
	if ncfg.Get("database:port") != nil {
		conf.Database.Port = ncfg.Get("database:port").(int)
	} else {
		conf.Database.Port = 3306
	}

	// Database TLS - secure by default
	if ncfg.Get("database:useTLS") != nil {
		conf.Database.UseTLS = ncfg.Get("database:useTLS").(bool)
	} else {
		conf.Database.UseTLS = true // ✅ Secure by default
	}

	// Ratelimit configuration with secure defaults
	if ncfg.Get("ratelimit:enabled") != nil {
		conf.RateLimit.Enabled = ncfg.Get("ratelimit:enabled").(bool)
	}
	//else {
	//	conf.RateLimit.Enabled = true // ✅ Enabled by default
	//}

	if ncfg.Get("ratelimit:ipMaxAttempts") != nil {
		conf.RateLimit.IPMaxAttempts = ncfg.Get("ratelimit:ipMaxAttempts").(int)
	}
	//else {
	//	conf.RateLimit.IPMaxAttempts = 10 // Default: 10 attempts
	//}

	if ncfg.Get("ratelimit:ipBlockMinutes") != nil {
		conf.RateLimit.IPBlockMinutes = ncfg.Get("ratelimit:ipBlockMinutes").(int)
	}
	//else {
	//	conf.RateLimit.IPBlockMinutes = 15 // Default: 15 minutes
	//}

	if ncfg.Get("ratelimit:usernameMaxAttempts") != nil {
		conf.RateLimit.UsernameMaxAttempts = ncfg.Get("ratelimit:usernameMaxAttempts").(int)
	}
	//else {
	//	conf.RateLimit.UsernameMaxAttempts = 5 // Default: 5 attempts
	//}

	if ncfg.Get("ratelimit:usernameBlockMinutes") != nil {
		conf.RateLimit.UsernameBlockMinutes = ncfg.Get("ratelimit:usernameBlockMinutes").(int)
	}
	//else {
	//	conf.RateLimit.UsernameBlockMinutes = 30 // Default: 30 minutes
	//}

	return conf
}

// getValuePair split and return parameter name and value in a slice of string
func getValuePair(s string, sep string) []string {
	nvPair := strings.Split(s, sep)
	return []string{strings.Trim(nvPair[0], " "), strings.Trim(nvPair[1], " ")}
}

// Add a value to configValues
func (s *configValues) Add(k string, v interface{}) {
	r := *s
	r[k] = v
}

// Get a value from configValues
func (s *configValues) Get(k string) interface{} {
	r := *s
	return r[k]
}
