package wgorm

import "fmt"

type Config struct {
	Host          string
	Port          int
	DBName        string
	MaxPool       int
	MinPool       int
	Username      string
	Password      string
	EnableTracing bool
}

func (c *Config) GetConnectionString() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d",
		c.Host,
		c.Username,
		c.Password,
		c.DBName,
		c.Port,
	)
}
