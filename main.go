// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package main

import (
	"encoding/base64"
	_ "expvar"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"gopkg.in/tylerb/graceful.v1"

	"github.com/DSchalla/go-pid"
	"github.com/Sirupsen/logrus"
	"github.com/TF2Stadium/Helen/config"
	"github.com/TF2Stadium/Helen/config/stores"
	"github.com/TF2Stadium/Helen/controllers/broadcaster"
	chelpers "github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	"github.com/TF2Stadium/Helen/controllers/socket"
	"github.com/TF2Stadium/Helen/database"
	"github.com/TF2Stadium/Helen/database/migrations"
	"github.com/TF2Stadium/Helen/helpers"
	_ "github.com/TF2Stadium/Helen/helpers/authority" // to register authority types
	_ "github.com/TF2Stadium/Helen/internal/pprof"    // to setup expvars
	"github.com/TF2Stadium/Helen/models"
	"github.com/TF2Stadium/Helen/models/event"
	"github.com/TF2Stadium/Helen/routes"
	socketServer "github.com/TF2Stadium/Helen/routes/socket"
	"github.com/TF2Stadium/etcd"
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/rs/cors"
)

var flagGen = flag.Bool("genkey", false, "write a 32bit key for encrypting cookies the given file, and exit")

func main() {
	helpers.InitLogger()

	flag.Parse()
	if *flagGen {
		key := securecookie.GenerateRandomKey(64)
		if len(key) == 0 {
			logrus.Fatal("Couldn't generate random key")
		}

		base64Key := base64.StdEncoding.EncodeToString(key)
		fmt.Println(base64Key)
		return
	}

	config.SetupConstants()

	if config.Constants.ProfilerAddr != "" {
		go graceful.Run(config.Constants.ProfilerAddr, 1*time.Second, nil)
		logrus.Info("Running Profiler at ", config.Constants.ProfilerAddr)
	}

	if config.Constants.EtcdAddr != "" {
		err := etcd.ConnectEtcd(config.Constants.EtcdAddr)
		if err != nil {
			logrus.Fatal(err)
		}

		node, err := etcd.SetAddr(config.Constants.EtcdService, config.Constants.RPCAddr)
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Info("Wrote key ", node.Key, "=", node.Value)
	}

	helpers.ConnectAMQP()
	event.StartListening()
	broadcaster.StartListening()

	helpers.InitAuthorization()
	database.Init()
	migrations.Do()
	stores.SetupStores()
	err := models.LoadLobbySettingsFromFile("assets/lobbySettingsData.json")
	if err != nil {
		logrus.Fatal(err)
	}

	models.ConnectRPC()
	models.DeleteUnusedServerRecords()

	chelpers.InitGeoIPDB()
	if config.Constants.SteamIDWhitelist != "" {
		go chelpers.WhitelistListener()
	}
	// lobby := models.NewLobby("cp_badlands", 10, "a", "a", 1)

	mux := http.NewServeMux()
	routes.SetupHTTP(mux)
	socket.RegisterHandlers()

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   config.AllowedCorsOrigins,
		AllowCredentials: true,
	}).Handler(context.ClearHandler(mux))

	pid := &pid.Instance{}
	if pid.Create() == nil {
		defer pid.Remove()
	}

	// start the server
	server := graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:         config.Constants.ListenAddress,
			Handler:      corsHandler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},

		ShutdownInitiated: func() {
			logrus.Info("Received SIGINT/SIGTERM, closing RPC")
			logrus.Info("waiting for socket requests to complete.")
			socketServer.Wait()
		},
	}

	logrus.Info("Serving on ", config.Constants.ListenAddress)
	err = server.ListenAndServe()
	if err != nil {
		logrus.Fatal(err)
	}
}
