// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gnuflag"
	"net/http"
	"strings"
)

type platformAdd struct {
	name       string
	dockerfile string
	fs         *gnuflag.FlagSet
}

func (p *platformAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "platform-add",
		Usage:   "platform-add <platform name> [--dockerfile/-d Dockerfile]",
		Desc:    "Add new platform to tsuru.",
		MinArgs: 1,
	}
}

func (p *platformAdd) Run(context *cmd.Context, client *cmd.Client) error {
	name := context.Args[0]
	body := fmt.Sprintf("name=%s&dockerfile=%s", name, p.dockerfile)
	url, err := cmd.GetURL("/platforms/add")
	request, err := http.NewRequest("PUT", url, strings.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	_, err = client.Do(request)
	if err != nil {
		return err
	}

	fmt.Fprintf(context.Stdout, "Platform successfully added!\n")
	return nil
}

func (p *platformAdd) Flags() *gnuflag.FlagSet {
	message := "The dockerfile url to create a platform"
	if p.fs == nil {
		p.fs = gnuflag.NewFlagSet("platform-add", gnuflag.ExitOnError)
		p.fs.StringVar(&p.dockerfile, "dockerfile", "", message)
		p.fs.StringVar(&p.dockerfile, "d", "", message)
	}

	return p.fs
}
