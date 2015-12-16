package controllerhelpers

import (
	"net"
	"strings"

	"github.com/TF2Stadium/Helen/config"
	"github.com/TF2Stadium/Helen/helpers"
	"github.com/oschwald/geoip2-golang"
)

var geodb *geoip2.Reader

func InitGeoIPDB() {
	if config.Constants.GeoIP == "" {
		return
	}

	var err error
	geodb, err = geoip2.Open(config.Constants.GeoIP)

	if err != nil {
		helpers.Logger.Fatal(err.Error())
	}
}

func GetRegion(server string) (string, string) {
	if config.Constants.GeoIP == "" {
		return "", ""
	}

	arr := strings.Split(server, ":")
	addr, err := net.ResolveIPAddr("ip4", arr[0])
	if err != nil {
		helpers.Logger.Error(err.Error())
		return "", ""
	}

	record, err := geodb.Country(addr.IP)
	if err != nil {
		helpers.Logger.Error(err.Error())
		return "", ""
	}

	if record.Country.Names["en"] == "Russia" {
		return "ru", "Russia"
	}

	return strings.ToLower(record.Continent.Code), record.Continent.Names["en"]
}
