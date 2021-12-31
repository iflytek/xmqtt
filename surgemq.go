// Copyright (c) 2014 The SurgeMQ Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"xmqtt/service"
	"xmqtt/utils/log"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "xmqtt"
	app.Usage = "xmqtt Server"
	app.Version = xmqttVERSION

	app.Action = func(c *cli.Context) error {
		s := new(service.MqttXSFServer)
		s.Init()
		s.Run()
		select {}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("main.main | app.Run err:%v", err)
	}
}
