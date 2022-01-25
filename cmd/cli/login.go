// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
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
	"crypto/sha512"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"

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
	loginOpts               loginOptions
)

type loginOptions struct {
	server    string
	username  string
	password  string
	keyPath   string
	stdinPass bool
}

type passReader func() ([]byte, error)

// NewLoginCmd returns login command
func NewLoginCmd() *cobra.Command {
	// loginCmd represents the "login" command
	loginCmd := &cobra.Command{
		Use:     "login SERVER [FLAGS]",
		Short:   "Login to an image registry",
		Example: loginExample,
		RunE:    loginCommand,
	}

	loginCmd.PersistentFlags().StringVarP(&loginOpts.username, "username", "u", "", "Username to access registry")
	loginCmd.PersistentFlags().BoolVarP(&loginOpts.stdinPass, "password-stdin", "p", false, "Read password from stdin")

	return loginCmd
}

func loginCommand(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errEmptyRegistry
	}
	if len(args) > 1 {
		return errTooManyArgs
	}
	if err := getRegistry(args); err != nil {
		return err
	}
	loginOpts.keyPath = util.DefaultRSAKeyPath

	if err := checkAuthOpt(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	msg, err := runLogin(ctx, cli, c)
	fmt.Println(msg)
	if err != nil {
		return err
	}
	return nil
}

func runLogin(ctx context.Context, cli Cli, c *cobra.Command) (string, error) {
	req, err := genLoginReq(c, false)
	if err != nil {
		return "", err
	}
	resp, err := cli.Client().Login(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "Failed to authenticate existing credentials") {
			fmt.Printf("Failed to authenticate existing credentials, please input auth info directly\n\n")
			if err = getAuthInfo(c); err != nil {
				return "", err
			}
			req, err = genLoginReq(c, true)
			if err != nil {
				return "", err
			}
			resp, err = cli.Client().Login(ctx, req)
			if err != nil {
				return loginFailed, err
			}
			return resp.Content, err
		}
		return loginFailed, err
	}

	return resp.Content, err
}

func checkAuthOpt() error {
	if loginOpts.stdinPass && loginOpts.username == "" {
		return errLackOfFlags
	}
	return nil
}

func getAuthInfo(c *cobra.Command) error {
	if err := getUsername(c); err != nil {
		return err
	}
	if err := getPassword(c); err != nil {
		return err
	}

	return nil
}

func genLoginReq(c *cobra.Command, shouldGetAuthInfo bool) (*pb.LoginRequest, error) {
	// first check auth info from auth.json, so no auth info
	// should be send from client to server
	if loginOpts.username == "" && loginOpts.password == "" {
		fmt.Printf("try to login with existing credentials...\n\n")
		return &pb.LoginRequest{
			Server:   loginOpts.server,
			Username: "",
			Password: "",
		}, nil
	}

	// if shouldGetAuthInfo is false, we don't need to getAuthInfo again
	// because this action will do in outer place
	if shouldGetAuthInfo || loginOpts.username != "" {
		if err := getAuthInfo(c); err != nil {
			return nil, err
		}
	}
	if err := encryptOpts(loginOpts.keyPath); err != nil {
		return nil, err
	}

	return &pb.LoginRequest{
		Server:   loginOpts.server,
		Username: loginOpts.username,
		Password: loginOpts.password,
	}, nil
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
	// in this scenario, it is second time trying to get username
	// if already got it, there is no need to get username again
	if loginOpts.username != "" {
		return nil
	}
	username, err := c.Flags().GetString("username")
	if err != nil {
		return err
	}

	if loginOpts.username == "" {
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
	// in this scenario, it is second time trying to get password
	// if already got it, there is no need to get pass again
	if loginOpts.password != "" {
		return nil
	}

	if loginOpts.stdinPass {
		if err := getPassFromStdin(os.Stdin); err != nil {
			return err
		}
	} else {
		r := func() ([]byte, error) {
			return term.ReadPassword(0)
		}
		if err := getPassFromInput(r); err != nil {
			return err
		}
	}
	return nil
}

func getPassFromInput(f passReader) error {
	fmt.Print("Password: ")
	termPass, err := f()
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

func getPassFromStdin(r io.Reader) error {
	var buf strings.Builder
	passScanner := bufio.NewScanner(r)
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

func encryptOpts(path string) error {
	key, err := util.ReadPublicKey(path)
	if err != nil {
		return err
	}
	encryptedPass, enErr := util.EncryptRSA(loginOpts.password, key, sha512.New())
	if enErr != nil {
		loginOpts.password = ""
		return enErr
	}

	loginOpts.password = encryptedPass
	return nil
}
