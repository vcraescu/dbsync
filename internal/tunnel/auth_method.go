package tunnel

import (
	"io/ioutil"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func newSSHAgent() (agent.Agent, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	return agent.NewClient(conn), nil
}

func readPKFile(file string) ([]byte, error) {
	buff, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return buff, nil
}

// CreateSSHAgentAuthMethod - creates new ssh agent auth method
func CreateSSHAgentAuthMethod() (ssh.AuthMethod, error) {
	ag, err := newSSHAgent()
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeysCallback(ag.Signers), nil
}

// CreatePKAuthMethod - creates a private key auth method
func CreatePKAuthMethod(file string) (ssh.AuthMethod, error) {
	buff, err := readPKFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buff)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(key), nil
}
