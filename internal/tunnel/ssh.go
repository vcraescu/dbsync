package tunnel

import (
	"fmt"
	"net"
	"io"
	"errors"
	"log"

	"golang.org/x/crypto/ssh"
)

type Endpoint struct {
	Host string
	Port int
	User string
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type SSHTunnel struct {
	local  Endpoint
	server Endpoint
	remote Endpoint
	cfg    *ssh.ClientConfig
}

func (tunnel *SSHTunnel) LocalPort() int {
	return tunnel.local.Port
}

func (tunnel *SSHTunnel) LocalHost() string {
	return tunnel.local.Host
}

func (tunnel *SSHTunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			err := tunnel.forward(conn)
			if err != nil {
				conn.Close()
				log.Panic(fmt.Sprintf("SSH Tunnel: %s", err))
			}
		}()
	}
}

func (tunnel *SSHTunnel) forward(localConn net.Conn) error {
	serverConn, err := ssh.Dial("tcp", tunnel.server.String(), tunnel.cfg)
	if err != nil {
		return errors.New(fmt.Sprintf("Server dial error: %s", err))
	}

	remoteConn, err := serverConn.Dial("tcp", tunnel.remote.String())
	if err != nil {
		return errors.New(fmt.Sprintf("Remote dial error: %s", err))
	}

	copyConn := func(writer, reader net.Conn) error {
		_, err := io.Copy(writer, reader)
		if err != nil {
			return errors.New(fmt.Sprintf("io.Copy error: %s", err))
		}

		return nil
	}

	go func() {
		err := copyConn(localConn, remoteConn)
		if err != nil {
			log.Println(fmt.Sprintf("SSH Tunnel: %s", err))
		}
	}()

	go func() {
		err := copyConn(remoteConn, localConn)
		if err != nil {
			log.Println(fmt.Sprintf("SSH Tunnel: %s", err))
		}
	}()

	return nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func CreateSSHTunnel(localEndpoint, serverEndpoint, remoteEndpoint Endpoint, authMethod *ssh.AuthMethod) (*SSHTunnel, error) {
	localPort, err := getFreePort()
	if err != nil {
		return nil, err
	}

	localEndpoint.Port = localPort

	sshConfig := &ssh.ClientConfig{
		User: serverEndpoint.User,
		Auth: []ssh.AuthMethod{
			*authMethod,
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	return &SSHTunnel{
		cfg: sshConfig,
		local: localEndpoint,
		server: serverEndpoint,
		remote: remoteEndpoint,
	}, nil
}

func StartSSHTunnel(localEndpoint, serverEndpoint, remoteEndpoint Endpoint, authMethod *ssh.AuthMethod) (*SSHTunnel, error) {
	tunn, err := CreateSSHTunnel(localEndpoint, serverEndpoint, remoteEndpoint, authMethod)
	if err != nil {
		return nil, err
	}

	go func() {
		log.Panicf("SSH Tunnel: %s", tunn.Start())
	}()

	return tunn, nil
}
