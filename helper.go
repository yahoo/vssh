//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
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
