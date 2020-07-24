// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// iSula-Kits licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-06-02
// Description: This file is used for "login" command

package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	loginExample = `isula-build login dockerhub.io
cat creds.txt | isula-build login -u cooper -p mydockerhub.io`
	loginFailed = "\nLogin Failed\n"
	maxInputLen = 128
)

var (
	errReadUsernameFromTerm = errors.New("could not read username from terminal")
	errReadPassFromTerm     = errors.New("could not read password from terminal")
	errLenTooLong           = errors.New("length of input exceeded")
	errEmptyUsername        = errors.New("username can not be empty")
	errEmptyAuth            = errors.New("auth info can not be empty")
	errEmptyRegistry        = errors.New("empty registry found, please input one registry")
	errTooManyArgs          = errors.New("too many arguments, login only accepts 1 argument")
	errLackOfFlags          = errors.New("must provides --password-stdin with --username")
)

type loginOptions struct {
	server    string
	key       string
	username  string
	password  string
	stdinPass bool
}

var loginOpts loginOptions

// NewLoginCmd returns login command
func NewLoginCmd() *cobra.Command {
	// loginCmd represents the "login" command
	loginCmd := &cobra.Command{
		Use:     "login",
		Short:   "Login to an image registry",
		Example: loginExample,
		RunE:    loginCommand,
	}

	loginCmd.PersistentFlags().StringVarP(&loginOpts.username, "username", "u", "", "Username to access registry")
	loginCmd.PersistentFlags().BoolVarP(&loginOpts.stdinPass, "password-stdin", "p", false, "Read password from stdin")

	return loginCmd
}

func loginCommand(c *cobra.Command, args []string) error {
	if err := newLoginOptions(c, args); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}
	msg, err := runLogin(ctx, cli)
	fmt.Println(msg)
	if err != nil {
		return err
	}
	return nil
}

func runLogin(ctx context.Context, cli Cli) (string, error) {
	if err := encryptOpts(); err != nil {
		return "", err
	}
	req := &pb.LoginRequest{
		Server:   loginOpts.server,
		Username: loginOpts.username,
		Password: loginOpts.password,
		Key:      loginOpts.key,
	}
	resp, err := cli.Client().Login(ctx, req)
	if err != nil {
		return loginFailed, err
	}
	return resp.Content, err
}

func newLoginOptions(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errEmptyRegistry
	}
	if len(args) > 1 {
		return errTooManyArgs
	}

	if err := getRegistry(args); err != nil {
		return err
	}

	if err := getUsername(c); err != nil {
		return err
	}

	if err := getPassword(c); err != nil {
		return err
	}

	return nil
}

func getRegistry(args []string) error {
	server, err := util.ParseServer(args[0])
	if err != nil {
		return err
	}
	loginOpts.server = server
	return nil
}

func getUsername(c *cobra.Command) error {
	username, err := c.Flags().GetString("username")
	if err != nil {
		return err
	}

	if !c.Flag("username").Changed {
		fmt.Print("Username: ")
		if _, err := fmt.Scanln(&username); err != nil {
			return errReadUsernameFromTerm
		}
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return errEmptyUsername
	}
	if len(username) > maxInputLen {
		return errLenTooLong
	}
	loginOpts.username = username
	return nil
}

func getPassword(c *cobra.Command) error {
	if c.Flag("password-stdin").Changed && !c.Flag("username").Changed {
		return errLackOfFlags
	}

	if loginOpts.stdinPass {
		if err := getPassFromStdin(); err != nil {
			return err
		}
	} else {
		if err := getPassFromInput(); err != nil {
			return err
		}
	}
	return nil
}

func getPassFromInput() error {
	fmt.Print("Password: ")
	termPass, err := terminal.ReadPassword(0)
	if err != nil {
		return errReadPassFromTerm
	}
	if len(termPass) > maxInputLen {
		return errLenTooLong
	}
	loginOpts.password = string(termPass)
	// get new line
	fmt.Println()
	return nil
}

func getPassFromStdin() error {
	var buf strings.Builder
	passScanner := bufio.NewScanner(os.Stdin)
	for passScanner.Scan() {
		if _, err := fmt.Fprint(&buf, passScanner.Text()); err != nil {
			return err
		}
	}

	if len(buf.String()) > maxInputLen {
		return errLenTooLong
	}
	if len(buf.String()) == 0 {
		return errEmptyAuth
	}

	loginOpts.password = buf.String()
	return nil
}

func encryptOpts() error {
	oriKey, err := util.GenerateCryptoKey(util.CryptoKeyLen)
	if err != nil {
		loginOpts.password = ""
		return err
	}
	key, pbkErr := util.PBKDF2(oriKey, util.CryptoKeyLen, sha256.New)
	if pbkErr != nil {
		loginOpts.password = ""
		return pbkErr
	}
	encryptedPass, enErr := util.EncryptAES(loginOpts.password, key)
	if enErr != nil {
		loginOpts.password = ""
		return enErr
	}

	loginOpts.password = encryptedPass
	loginOpts.key = key
	return nil
}
