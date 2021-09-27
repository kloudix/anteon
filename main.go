/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"ddosify.com/hammer/config"
	"ddosify.com/hammer/core"
	"ddosify.com/hammer/core/types"
	"ddosify.com/hammer/core/util"
)

//TODO: what about -preview flag? Users can see how many requests will be sent per second with the given parameters.

const headerRegexp = `^([\w-]+):\s*(.+)`

// We might consider to use Viper: https://github.com/spf13/viper
var (
	reqCount = flag.Int("n", types.DefaultReqCount, "Total request count")
	duration = flag.Int("d", types.DefaultDuration, "Test duration in seconds")
	loadType = flag.String("l", types.DefaultLoadType, "Type of the load test [linear, incremental, waved]")

	protocol = flag.String("p", types.DefaultProtocol, "[HTTP, HTTPS]")
	method   = flag.String("m", types.DefaultMethod,
		"Request Method Type. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]")
	payload = flag.String("b", "", "Payload of the network packet")
	auth    = flag.String("a", "", "Basic authentication, username:password")
	headers header

	target  = flag.String("t", "", "Target URL")
	timeout = flag.Int("T", types.DefaultTimeout, "Request timeout in seconds")

	proxy  = flag.String("P", "", "Proxy address as host:port")
	output = flag.String("o", types.DefaultOutputType, "Output destination")

	configPath = flag.String("config", "",
		"Json config file path. If a config file is provided, other flag values will be ignored.")
)

func main() {
	flag.Var(&headers, "h", "Request Headers. Ex: -h 'Accept: text/html' -h 'Content-Type: application/xml'")
	flag.Parse()

	h, err := createHammer()

	if err != nil {
		exitWithMsg(err.Error())
	}

	if err := h.Validate(); err != nil {
		exitWithMsg(err.Error())
	}

	run(h)
}

func createHammer() (h types.Hammer, err error) {
	if *configPath != "" {
		h, err = createHammerFromConfigFile()
	} else {
		h, err = createHammerFromFlags()
	}
	return h, err
}

var createHammerFromConfigFile = func() (h types.Hammer, err error) {
	c, err := config.NewConfigReader(*configPath, config.ConfigTypeJson)
	if err != nil {
		return
	}

	h, err = c.CreateHammer()
	if err != nil {
		return
	}
	return
}

var run = func(h types.Hammer) {
	ctx, cancel := context.WithCancel(context.Background())

	engine, err := core.NewEngine(ctx, h)
	if err != nil {
		exitWithMsg(err.Error())
	}

	err = engine.Init()
	if err != nil {
		exitWithMsg(err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	engine.Start()
}

var createHammerFromFlags = func() (h types.Hammer, err error) {
	if *target == "" {
		err = fmt.Errorf("Please provide the target url with -t flag")
		return
	}

	s, err := createScenario()
	if err != nil {
		return
	}

	p, err := createProxy()
	if err != nil {
		return
	}

	h = types.Hammer{
		TotalReqCount:     *reqCount,
		LoadType:          strings.ToLower(*loadType),
		TestDuration:      *duration,
		Scenario:          s,
		Proxy:             p,
		ReportDestination: *output,
	}
	return
}

func createProxy() (p types.Proxy, err error) {
	var proxyURL *url.URL
	if *proxy != "" {
		proxyURL, err = url.Parse(*proxy)
		if err != nil {
			return
		}
	}

	p = types.Proxy{
		Strategy: types.ProxyTypeSingle,
		Addr:     proxyURL,
	}
	return
}

func createScenario() (s types.Scenario, err error) {
	// Auth
	var a types.Auth
	if *auth != "" {
		creds := strings.Split(*auth, ":")
		if len(creds) != 2 {
			err = fmt.Errorf("auth credentials couldn't be parsed")
			return
		}

		a = types.Auth{
			Type:     types.AuthHttpBasic,
			Username: creds[0],
			Password: creds[1],
		}
	}

	// Protocol & URL
	url, err := util.StrToURL(*protocol, *target)
	if err != nil {
		return
	}

	h, err := parseHeaders(headers)
	if err != nil {
		return
	}

	s = types.Scenario{
		Scenario: []types.ScenarioItem{
			{
				ID:       1,
				Protocol: strings.ToUpper(url.Scheme),
				Method:   strings.ToUpper(*method),
				Auth:     a,
				Headers:  h,
				Payload:  *payload,
				URL:      url.String(),
				Timeout:  *timeout,
			},
		},
	}

	return
}

func exitWithMsg(msg string) {
	if msg != "" {
		msg = "err: " + msg
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

func parseHeaders(headersArr []string) (headersMap map[string]string, err error) {
	re := regexp.MustCompile(headerRegexp)
	headersMap = make(map[string]string)
	for _, h := range headersArr {
		matches := re.FindStringSubmatch(h)
		if len(matches) < 1 {
			err = fmt.Errorf("invalid header:  %v", h)
			return
		}
		headersMap[matches[1]] = matches[2]
	}
	return
}

type header []string

func (h *header) String() string {
	return fmt.Sprintf("%s - %d", *h, len(*h))
}

func (h *header) Set(value string) error {
	*h = append(*h, value)
	return nil
}
