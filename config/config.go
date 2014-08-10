package config

import toml "github.com/BurntSushi/toml"

type Config struct {
	Host    string `toml:"host"`
	Port    string `toml:"port"`
	Key     string `toml:"key"`
	Secret  string `toml:"secret"`
	Timeout string `toml:"timeout"`
	Crawl   bool   `toml:"crawl"`

	TownUser     string `toml:"townuser"`
	TownPassword string `toml:"townpassword"`

	GhostUser     string `toml:"ghostuser"`
	GhostPassword string `toml:"ghostpassword"`

	DBUser     string `toml:"dbuser"`
	DBPassword string `toml:"dbpassword"`
	DBName     string `toml:"dbdatabase"`

	CacheSize int `toml:"cachesize"`
	CacheFree int `toml:"cachefree"`
}

func Load(file string) (c *Config, err error) {
	if _, err := toml.DecodeFile(file, &c); err != nil {
		return nil, err
	}
	return
}
