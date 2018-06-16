package cmd

import (
	"fmt"
	"os"
	"log"
	"errors"

	"github.com/spf13/cobra"
	"github.com/vcraescu/dbsync/internal/database/mysql"
	"github.com/vcraescu/dbsync/internal/tunnel"
	"golang.org/x/crypto/ssh"
	"github.com/vcraescu/dbsync/internal/net"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const Version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:          "dbsync master_server slave_server",
	Short:        "Sync 2 MySQL databases",
	Long:         `Sync 2 MySQL databases`,
	Version:      Version,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		masterName := args[0]
		slaveName := args[1]
		cfg, ok := config.Servers[masterName]
		if !ok {
			return errors.New("master server name not found in config file")
		}
		config.Master = cfg

		cfg, ok = config.Servers[slaveName]
		if !ok {
			return errors.New("slave server name not found in config file")
		}
		config.Slave = cfg

		if !config.Validate() {
			return errors.New("invalid config")
		}

		return nil
	},
}

type SSHConfig struct {
	Host string `mapstructure:"host"`
	User string `mapstructure:"user"`
	Port int    `mapstructure:"port"`
	Key  string `mapstructure:"key"`
}

type ServerConfig struct {
	SSHConfig SSHConfig `mapstructure:"ssh"`
	Username  string    `mapstructure:"username"`
	Password  string    `mapstructure:"password"`
	Host      string    `mapstructure:"host"`
	Schema    string    `mapstructure:"schema"`
	Port      int       `mapstructure:"port"`
}

type Config struct {
	Servers    map[string]ServerConfig `mapstructure:"servers"`
	Master     ServerConfig
	Slave      ServerConfig
	configFile string
}

var config = &Config{}

func (cfg *Config) CreateMasterConnectionConfig() *mysql.ConnectionConfig {
	ip, err := cfg.GetMasterHostIP()
	if err != nil {
		panic(err)
	}

	return &mysql.ConnectionConfig{
		Username: cfg.Master.Username,
		Password: cfg.Master.Password,
		Host:     ip,
		Port:     cfg.Master.Port,
		Schema:   cfg.Master.Schema,
	}
}

func (cfg *Config) CreateSlaveConnectionConfig() *mysql.ConnectionConfig {
	ip, err := cfg.GetSlaveHostIP()
	if err != nil {
		panic(err)
	}

	return &mysql.ConnectionConfig{
		Username: cfg.Slave.Username,
		Password: cfg.Slave.Password,
		Host:     ip,
		Port:     cfg.Slave.Port,
		Schema:   cfg.Slave.Schema,
	}
}

func (cfg *Config) SlaveSSHTunnelIsRequired() bool {
	return cfg.Slave.SSHConfig.User != "" && cfg.Slave.SSHConfig.Host != "" && cfg.Slave.SSHConfig.Port > 0
}

func (cfg *Config) MasterSSHTunnelIsRequired() bool {
	return cfg.Master.SSHConfig.User != "" && cfg.Master.SSHConfig.Host != "" && cfg.Master.SSHConfig.Port > 0
}

func (cfg *Config) GetMasterHostIP() (string, error) {
	return net.HostnameToIP4(cfg.Master.Host)
}

func (cfg *Config) GetSlaveHostIP() (string, error) {
	return net.HostnameToIP4(cfg.Slave.Host)
}

func (cfg *Config) Validate() bool {
	valid := true
	if cfg.Master.Username == "" {
		fmt.Fprintln(os.Stderr, "Error: Master username is required")
		valid = false
	}

	if cfg.Master.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: Master password is required")
		valid = false
	}

	if cfg.Master.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Master host is required")
		valid = false
	}

	if cfg.Master.Port <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Master port is invalid")
		valid = false
	}

	if cfg.Master.Schema == "" {
		fmt.Fprintln(os.Stderr, "Error: Master schema is required")
		valid = false
	}

	if cfg.Slave.Username == "" {
		fmt.Fprintln(os.Stderr, "Error: Master username is required")
		valid = false
	}

	if cfg.Slave.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: Master password is required")
		valid = false
	}

	if cfg.Slave.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Master host is required")
		valid = false
	}

	if cfg.Slave.Port <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Slave port is invalid")
		valid = false
	}

	if cfg.Slave.Schema == "" {
		fmt.Fprintln(os.Stderr, "Error: Slave schema is required")
		valid = false
	}

	if cfg.Slave.SSHConfig.User != "" || cfg.Slave.SSHConfig.Host != "" || cfg.Slave.SSHConfig.Port > 0 {
		if cfg.Slave.SSHConfig.User == "" {
			fmt.Fprintln(os.Stderr, "Error: Slave SSH user is required")
			valid = false
		}

		if cfg.Slave.SSHConfig.Host == "" {
			fmt.Fprintln(os.Stderr, "Error: Slave SSH host is required")
			valid = false
		}

		if cfg.Slave.SSHConfig.Port <= 0 {
			fmt.Fprintln(os.Stderr, "Error: Slave SSH port is invalid")
			valid = false
		}
	}

	return valid
}

func (cfg *SSHConfig) CreateAuthMethod() (*ssh.AuthMethod, error) {
	if cfg.Key == "" {
		authMethod, err := tunnel.CreateSSHAgentAuthMethod()
		if err != nil {
			return nil, fmt.Errorf("ssh agent auth method: %s", err)
		}
		log.Println("Authenticating to slave server using SSH Agent")

		return &authMethod, nil
	}

	file, err := homedir.Expand(cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("error expanding home dir: %s", err)
	}

	authMethod, err := tunnel.CreatePKAuthMethod(file)
	if err != nil {
		return nil, fmt.Errorf("public key auth method: %s", err)
	}

	log.Printf("Authenticating to slave server using private key: %s\n", file)

	return &authMethod, nil
}

func startSSHTunnel(mysqlCfg *mysql.ConnectionConfig, sshCfg SSHConfig) (*tunnel.SSHTunnel, error) {
	localEndpoint := tunnel.Endpoint{
		Host: "localhost",
	}
	serverEndpoint := tunnel.Endpoint{
		Host: sshCfg.Host,
		Port: config.Slave.SSHConfig.Port,
		User: config.Slave.SSHConfig.User,
	}
	remoteEndpoint := tunnel.Endpoint{
		Host: mysqlCfg.Host,
		Port: mysqlCfg.Port,
	}

	authMethod, err := sshCfg.CreateAuthMethod()
	if err != nil {
		return nil, err
	}

	t, err := tunnel.StartSSHTunnel(localEndpoint, serverEndpoint, remoteEndpoint, authMethod)
	if err != nil {
		return nil, err
	}

	mysqlCfg.Port = t.LocalPort()

	return t, nil
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&config.configFile,
		"config",
		"",
		"Config file (default $HOME/.dbsync.yaml or ./.dbsync.yml)",
	)

	rootCmd.AddCommand(syncCmd)
}

func initConfig() {
	if config.configFile != "" {
		viper.SetConfigFile(config.configFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal("Can't read config:", err)
		}

		viper.Unmarshal(&config)
		return
	}

	viper.SetConfigName(".dbsync")
	viper.AddConfigPath("$HOME/.dbsync")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return
	}

	viper.Unmarshal(&config)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
