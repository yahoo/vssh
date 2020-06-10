//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

// GetConfigPEM returns SSH configuration that uses the the
// public keys method for remote authentication
func GetConfigPEM(user, keyFile string) (*ssh.ClientConfig, error) {

	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}
	auths := []ssh.AuthMethod{ssh.PublicKeys(signer)}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return config, nil
}

// GetConfigUserPass returns SSH configuration that uses the given
// username and password
func GetConfigUserPass(user, password string) *ssh.ClientConfig {
	auths := []ssh.AuthMethod{ssh.Password(password)}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return config
}
