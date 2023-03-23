//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io/ioutil"
	"net"
)

// GetConfigPEM returns SSH configuration that uses the given private key.
// the keyfile should be unencrypted PEM-encoded private key file.
func GetConfigPEM(user, keyFile string) (*ssh.ClientConfig, error) {
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	return getConfig(user, ssh.PublicKeys(signer)), nil
}

// GetConfigPEMWithPassphrase returns SSH configuration that uses the given private key and passphrase.
func GetConfigPEMWithPassphrase(user, keyFile string, passphrase string) (*ssh.ClientConfig, error) {
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	return getConfig(user, ssh.PublicKeys(signer)), nil
}

// GetConfigSSHAgent returns SSH configuration from SSH Agent.
// default socket can get from env: os.Getenv("SSH_AUTH_SOCK")
func GetConfigSSHAgent(user, socket string) (*ssh.ClientConfig, error) {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to ssh agent: %v", err)
	}
	agentconn := agent.NewClient(conn)
	signers, err := agentconn.Signers()
	if err != nil {
		return nil, fmt.Errorf("unable to get signers: %v", err)
	}

	return getConfig(user, ssh.PublicKeys(signers...)), nil
}

// GetConfigUserPass returns SSH configuration that uses the given
// username and password.
func GetConfigUserPass(user, password string) *ssh.ClientConfig {
	return getConfig(user, ssh.Password(password))
}

func getConfig(user string, auths ...ssh.AuthMethod) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
