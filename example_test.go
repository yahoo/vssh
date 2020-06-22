//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/yahoo/vssh"
)

// This example demonstrates the use of stream
func ExampleStream() {
	vs := vssh.New().Start()
	config, _ := vssh.GetConfigPEM("ubuntu", "aws.pem")
	vs.AddClient("54.193.17.197:22", config, vssh.SetMaxSessions(4))
	vs.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := "uname -a"
	timeout, _ := time.ParseDuration("5s")
	respChan := vs.Run(ctx, cmd, timeout)

	resp := <-respChan
	if err := resp.Err(); err != nil {
		log.Fatal(err)
	}

	stream := resp.GetStream()
	defer stream.Close()

	for stream.ScanStdout() {
		txt := stream.TextStdout()
		fmt.Println(txt)
	}

	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}
}

// This example demonstrates the use of GetText() for two hosts.
func ExampleResponse() {
	// construct and start the vssh
	vs := vssh.New().Start()

	// create ssh configuration with user/pass
	// you can create this configuration by golang ssh package
	config := vssh.GetConfigUserPass("vssh", "vssh")

	// add clients to vssh with one option: max session
	// there are other options that you can add to this method
	for _, addr := range []string{"54.193.17.197:22", "192.168.2.19:22"} {
		vs.AddClient(addr, config, vssh.SetMaxSessions(4))
	}

	// wait until vssh connected to all the clients
	vs.Wait()

	// create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// send the ping command to clients with 6 seconds timeout
	cmd := "ping -c 4 192.168.55.10"
	timeout, _ := time.ParseDuration("6s")
	respChan := vs.Run(ctx, cmd, timeout)

	// get the resp channel for each client
	for resp := range respChan {
		// in case of the connectivity issue to client
		if err := resp.Err(); err != nil {
			log.Println(err)
			continue
		}

		// get the returned data from client
		outTxt, errTxt, err := resp.GetText(vs)

		// check the error like timeout but still
		// we can have data on outTxt and errTxt
		if err != nil {
			log.Println(err)
		}

		// print command's stdout
		fmt.Println(outTxt)

		// print command's stderr
		fmt.Println(errTxt)

		// print exit status of the remote command
		fmt.Println(resp.ExitStatus())
	}
}

// This example demonstrates integration vSSH with AWS EC2
func Example_cloud() {
	vs := vssh.New().Start()
	config, _ := vssh.GetConfigPEM("ubuntu", "aws.pem")

	// AWS EC2 Golang SDK
	// Please check their website for more information
	// https://docs.aws.amazon.com/sdk-for-go/
	awsConfig := &aws.Config{
		Region:      aws.String("us-west-1"),
		Credentials: credentials.NewStaticCredentials("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		fmt.Println("error creating new session:", err.Error())
		log.Fatal(err)
	}
	ec2svc := ec2.New(sess)
	params := &ec2.DescribeInstancesInput{
		// filter running instances at us-west-1
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		fmt.Println("there was an error listing instances in", err.Error())
		log.Fatal(err.Error())
	}

	// iterate over the EC2 running instances and add to vssh
	for idx := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			labels := make(map[string]string)
			for _, tag := range inst.Tags {
				labels[*tag.Key] = *tag.Value
			}
			addr := net.JoinHostPort(*inst.PublicIpAddress, "22")
			vs.AddClient(addr, config, vssh.SetLabels(labels))
		}
	}

	vs.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := "uname -a"
	timeout, _ := time.ParseDuration("5s")
	respChan := vs.Run(ctx, cmd, timeout)

	for resp := range respChan {
		// in case of the connectivity issue to client
		if err := resp.Err(); err != nil {
			log.Println(err)
			continue
		}

		// get the returned data from client
		outTxt, errTxt, err := resp.GetText(vs)

		// check the error like timeout but still
		// we can have data on outTxt and errTxt
		if err != nil {
			log.Println(err)
		}

		// print command's stdout
		fmt.Println(outTxt)

		// print command's stderr
		fmt.Println(errTxt)

		// print exit status of the remote command
		fmt.Println(resp.ExitStatus())
	}
}

// This example demonstrates how to set limit the amount of returned data
func ExampleSetLimitReaderStdout() {

	// construct and start the vssh
	vs := vssh.New().Start()

	// create ssh configuration
	// you can create this configuration by golang ssh package
	config, _ := vssh.GetConfigPEM("ubuntu", "aws.pem")

	// add client to vssh with one option: max-sessions
	// there are other options that you can add to this method
	vs.AddClient("54.215.209.152:22", config, vssh.SetMaxSessions(2))

	// wait until vssh connected to the client
	vs.Wait()

	// create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout, _ := time.ParseDuration("6s")

	// run dmesg command with limit the amounth of returned data to 512 bytes
	respChan := vs.Run(ctx, "dmesg", timeout, vssh.SetLimitReaderStdout(1024))

	// get the resp
	resp := <-respChan
	// in case of the connectivity issue to client
	if err := resp.Err(); err != nil {
		log.Fatal(err)
	}

	outTxt, errTxt, err := resp.GetText(vs)
	// check the error like timeout but still
	// we can have data on outTxt and errTxt
	if err != nil {
		log.Println(err)
	}

	fmt.Println(outTxt)
	fmt.Println(errTxt)
}
