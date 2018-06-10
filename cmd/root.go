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

var rootCmd = &cobra.Command{
	Use:     "dbsync",
	Short:   "Sync 2 MySQL databases",
	Long:    `Sync 2 MySQL databases`,
	Version: "0.1",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !rootCmdFlags.Validate() {
			return errors.New("invalid arguments")
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

type RootFlags struct {
	Master struct {
		SSHCfg   SSHConfig `mapstructure:"ssh"`
		Username string    `mapstructure:"username"`
		Password string    `mapstructure:"password"`
		Host     string    `mapstructure:"host"`
		Schema   string    `mapstructure:"schema"`
		Port     int       `mapstructure:"port"`
	} `mapstructure:"master"`
	Slave struct {
		SSHCfg   SSHConfig `mapstructure:"ssh"`
		Username string    `mapstructure:"username"`
		Password string    `mapstructure:"password"`
		Host     string    `mapstructure:"host"`
		Schema   string    `mapstructure:"schema"`
		Port     int       `mapstructure:"port"`
	} `mapstructure:"slave"`

	configFile string
}

var rootCmdFlags = &RootFlags{}

func (f *RootFlags) CreateMasterConnectionConfig() *mysql.ConnectionConfig {
	ip, err := f.GetMasterHostIP()
	if err != nil {
		panic(err)
	}

	return &mysql.ConnectionConfig{
		Username: f.Master.Username,
		Password: f.Master.Password,
		Host:     ip,
		Port:     f.Master.Port,
		Schema:   f.Master.Schema,
	}
}

func (f *RootFlags) CreateSlaveConnectionConfig() *mysql.ConnectionConfig {
	ip, err := f.GetSlaveHostIP()
	if err != nil {
		panic(err)
	}

	return &mysql.ConnectionConfig{
		Username: f.Slave.Username,
		Password: f.Slave.Password,
		Host:     ip,
		Port:     f.Slave.Port,
		Schema:   f.Slave.Schema,
	}
}

func (f *RootFlags) SlaveSSHTunnelIsRequired() bool {
	return f.Slave.SSHCfg.User != "" && f.Slave.SSHCfg.Host != "" && f.Slave.SSHCfg.Port > 0
}

func (f *RootFlags) MasterSSHTunnelIsRequired() bool {
	return f.Master.SSHCfg.User != "" && f.Master.SSHCfg.Host != "" && f.Master.SSHCfg.Port > 0
}

func (f *RootFlags) GetMasterHostIP() (string, error) {
	return net.HostnameToIP4(f.Master.Host)
}

func (f *RootFlags) GetSlaveHostIP() (string, error) {
	return net.HostnameToIP4(f.Slave.Host)
}

func (f *RootFlags) Validate() bool {
	valid := true
	if f.Master.Username == "" {
		fmt.Fprintln(os.Stderr, "Error: Master username is required")
		valid = false
	}

	if f.Master.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: Master password is required")
		valid = false
	}

	if f.Master.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Master host is required")
		valid = false
	}

	if f.Master.Port <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Master port is invalid")
		valid = false
	}

	if f.Master.Schema == "" {
		fmt.Fprintln(os.Stderr, "Error: Master schema is required")
		valid = false
	}

	if f.Slave.Username == "" {
		fmt.Fprintln(os.Stderr, "Error: Master username is required")
		valid = false
	}

	if f.Slave.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: Master password is required")
		valid = false
	}

	if f.Slave.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Master host is required")
		valid = false
	}

	if f.Slave.Port <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Slave port is invalid")
		valid = false
	}

	if f.Slave.Schema == "" {
		fmt.Fprintln(os.Stderr, "Error: Slave schema is required")
		valid = false
	}

	if f.Slave.SSHCfg.User != "" || f.Slave.SSHCfg.Host != "" || f.Slave.SSHCfg.Port > 0 {
		if f.Slave.SSHCfg.User == "" {
			fmt.Fprintln(os.Stderr, "Error: Slave SSH user is required")
			valid = false
		}

		if f.Slave.SSHCfg.Host == "" {
			fmt.Fprintln(os.Stderr, "Error: Slave SSH host is required")
			valid = false
		}

		if f.Slave.SSHCfg.Port <= 0 {
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
		Port: rootCmdFlags.Slave.SSHCfg.Port,
		User: rootCmdFlags.Slave.SSHCfg.User,
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
		&rootCmdFlags.configFile,
		"config",
		"",
		"Config file (default $HOME/.dbsync.yaml or ./.dbsync.yml) - Note: all the other CLI arguments will be ignored",
	)

	rootCmd.PersistentFlags().StringVar(
		&rootCmdFlags.Master.Username,
		"master-user",
		"root",
		"master MySQL username",
	)

	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Master.Password, "master-password", "", "master MySQL password")
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Master.Host, "master-host", "localhost", "master MySQL host")
	rootCmd.PersistentFlags().IntVar(&rootCmdFlags.Master.Port, "master-port", 3306, "master MySQL port")
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Master.Schema, "master-schema", "", "master MySQL database")

	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Slave.Username, "slave-user", "root", "slave MySQL username")
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Slave.Password, "slave-password", "", "slave MySQL password")
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Slave.Host, "slave-host", "localhost", "slave MySQL host")
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Slave.Schema, "slave-schema", "", "slave MySQL database")
	rootCmd.PersistentFlags().IntVar(&rootCmdFlags.Slave.Port, "slave-port", 3306, "slave MySQL port")

	rootCmd.PersistentFlags().StringVar(
		&rootCmdFlags.Slave.SSHCfg.User,
		"slave-ssh-user",
		"",
		"creates a SSH tunnel for slave host connection",
	)
	rootCmd.PersistentFlags().StringVar(&rootCmdFlags.Slave.SSHCfg.Host, "slave-ssh-host", "", "SSH host for tunneling")
	rootCmd.PersistentFlags().IntVar(&rootCmdFlags.Slave.SSHCfg.Port, "slave-ssh-port", 22, "SSH port for tunneling")
	rootCmd.PersistentFlags().StringVar(
		&rootCmdFlags.Slave.SSHCfg.Key,
		"slave-ssh-key",
		"",
		"SSH private key path for ssh tunnel authentication method",
	)

	rootCmd.AddCommand(syncCmd)
}

func initConfig() {
	if rootCmdFlags.configFile != "" {
		viper.SetConfigFile(rootCmdFlags.configFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal("Can't read config:", err)
		}

		viper.Unmarshal(&rootCmdFlags)
		return
	}

	viper.SetConfigName(".dbsync")
	viper.AddConfigPath("$HOME/.dbsync")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return
	}

	viper.Unmarshal(&rootCmdFlags)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
