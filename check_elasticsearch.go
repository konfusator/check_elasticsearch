// Copyright (C) 2015 Michael Fischer v. Mollard 

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fractalcat/nagiosplugin"
	"io/ioutil"
	"net/http"
	"time"
)

type ClusterHealth struct {
	Cluster_name string
	Status string
	Timed_out bool
	Number_of_nodes int64
	Number_of_data_nodes int64
	Active_primary_shards int64
	Active_shards int64
	Relocating_shards int64
	Initializing_shards int64
	Unassigned_shards int64
}

func main() {
	var (
		host string
		port uint
		timeout int64
	)

	flag.StringVar(&host, "H", "127.0.0.1", "Target host")
	flag.UintVar(&port, "p", 9200, "Target port")
	flag.Int64Var(&timeout, "timeout", 10, "Plugin timeout")
	flag.Parse()

	check := nagiosplugin.NewCheck()

	// If we exit early or panic() we'll still output a result.
	defer check.Finish()

	// now for the timeout I don't really need the return values,
	// but it looks like something has to be written in the
	// channel
	c := make(chan int, 1)
	go func() { c <- cluster_health(check, host, port)} ()
	select {
	case <-c:
		return
	case <-time.After(time.Duration(timeout)*time.Second):
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("Timeout after %d seconds", timeout))
		return
	}

}

func cluster_health(check *nagiosplugin.Check, host string, port uint) int {
	// Read the cluster_health API
	url := fmt.Sprintf("http://%s:%d/_cluster/health", host, port)
	res, err := http.Get(url)
	if err != nil {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("error: %v", err))
		return int(nagiosplugin.CRITICAL)
	}
	if res.StatusCode != 200 {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("Unexpected Status %d for %s",
			res.StatusCode, url))
		return int(nagiosplugin.CRITICAL)
	}
	
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("read error: %v", err))
		return int(nagiosplugin.UNKNOWN)
	}

	var data ClusterHealth
	err = json.Unmarshal(body, &data)
	if err != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("JSON error: %v", err))
		return int(nagiosplugin.UNKNOWN)
	}
	check.AddPerfDatum("number_of_nodes", "", float64(data.Number_of_nodes))
	check.AddPerfDatum("number_of_data_nodes", "", float64(data.Number_of_data_nodes))
	check.AddPerfDatum("active_primary_shards", "", float64(data.Active_primary_shards))
	check.AddPerfDatum("active_shards", "", float64(data.Active_shards))
	check.AddPerfDatum("unassigned_shards", "", float64(data.Unassigned_shards))
	switch data.Status{
	case "green":
		check.AddResult(nagiosplugin.OK,
			fmt.Sprintf("Cluster '%s': Status is green ", data.Cluster_name))
		return int(nagiosplugin.OK)
	case "yellow":
		check.AddResult(nagiosplugin.WARNING,
			fmt.Sprintf("Cluster '%s': Status is yellow", data.Cluster_name))
		return int(nagiosplugin.WARNING)
		
	case "red":
		check.AddResult(nagiosplugin.CRITICAL,
			fmt.Sprintf("Cluster '%s': Status is red", data.Cluster_name))
		return int(nagiosplugin.CRITICAL)
	}
	return int(nagiosplugin.UNKNOWN)
}