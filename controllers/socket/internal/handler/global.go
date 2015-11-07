// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package handler

import (
	"encoding/json"

	chelpers "github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	"github.com/TF2Stadium/Helen/models"
	"github.com/TF2Stadium/wsevent"
	"github.com/bitly/go-simplejson"
)

func GetConstant(server *wsevent.Server, so *wsevent.Client, data string) string {
	var args struct {
		Constant string `json:"constant"`
	}
	if err := chelpers.GetParams(data, &args); err != nil {
		bytes, _ := chelpers.BuildFailureJSON(err.Error(), -1).Encode()
		return string(bytes)
	}

	output := simplejson.New()
	switch args.Constant {
	case "lobbySettingsList":
		output = models.LobbySettingsToJson()
	default:
		bytes, _ := chelpers.BuildFailureJSON("Unknown constant.", -1).Encode()
		return string(bytes)
	}

	outputString, err := output.Encode()
	if err != nil {
		bytes, _ := chelpers.BuildFailureJSON(err.Error(), -1).Encode()
		return string(bytes)
	}

	var resp struct {
		Success bool   `json:"success"`
		Data    string `json:"data"`
	}
	resp.Success = true
	resp.Data = string(outputString)

	bytes, _ := json.Marshal(resp)
	return string(bytes)
}
