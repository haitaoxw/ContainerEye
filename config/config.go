package config

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	
	Database struct {
		Path string `yaml:"path"` // SQLite database file path
	} `yaml:"database"`
	
	Alert struct {
		Slack struct {
			Token   string `yaml:"token"`
			Channel string `yaml:"channel"`
		} `yaml:"slack"`
		
		Email struct {
			SMTPHost     string   `yaml:"smtp_host"`
			SMTPPort     int      `yaml:"smtp_port"`
			From         string   `yaml:"from"`
			Password     string   `yaml:"password"`
			ToReceivers  []string `yaml:"to_receivers"`
		} `yaml:"email"`
	} `yaml:"alert"`
}
