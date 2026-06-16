package access_setting

import (
	"net"
	"os"
	"strings"
	"sync"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/oschwald/maxminddb-golang"
)

type CountryLookupResult struct {
	CountryCode string
	Source      string
	Known       bool
}

type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	RegisteredCountry struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"registered_country"`
	RepresentedCountry struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"represented_country"`
}

var (
	geoIPMu     sync.Mutex
	geoIPReader *maxminddb.Reader
	geoIPPath   string
)

var euCountryCodes = map[string]struct{}{
	"AT": {},
	"BE": {},
	"BG": {},
	"HR": {},
	"CY": {},
	"CZ": {},
	"DK": {},
	"EE": {},
	"FI": {},
	"FR": {},
	"DE": {},
	"GR": {},
	"HU": {},
	"IE": {},
	"IT": {},
	"LV": {},
	"LT": {},
	"LU": {},
	"MT": {},
	"NL": {},
	"PL": {},
	"PT": {},
	"RO": {},
	"SK": {},
	"SI": {},
	"ES": {},
	"SE": {},
}

func NormalizeCountryCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func IsChinaMainlandCountryCode(code string) bool {
	return NormalizeCountryCode(code) == "CN"
}

func IsEuropeanUnionCountryCode(code string) bool {
	_, ok := euCountryCodes[NormalizeCountryCode(code)]
	return ok
}

func LookupCountry(ip net.IP) CountryLookupResult {
	if ip == nil {
		return CountryLookupResult{}
	}

	path := strings.TrimSpace(accessControlSetting.GeoIPDatabasePath)
	if path == "" {
		return CountryLookupResult{}
	}

	reader, ok := getGeoIPReader(path)
	if !ok {
		return CountryLookupResult{}
	}

	var record countryRecord
	if err := reader.Lookup(ip, &record); err != nil {
		common.SysLog("geoip lookup failed: " + err.Error())
		return CountryLookupResult{}
	}

	code := firstCountryCode(
		record.Country.ISOCode,
		record.RegisteredCountry.ISOCode,
		record.RepresentedCountry.ISOCode,
	)
	if code == "" {
		return CountryLookupResult{}
	}

	return CountryLookupResult{
		CountryCode: code,
		Source:      "mmdb",
		Known:       true,
	}
}

func getGeoIPReader(path string) (*maxminddb.Reader, bool) {
	geoIPMu.Lock()
	defer geoIPMu.Unlock()

	if geoIPReader != nil && geoIPPath == path {
		return geoIPReader, true
	}

	if geoIPReader != nil {
		_ = geoIPReader.Close()
		geoIPReader = nil
		geoIPPath = ""
	}

	if _, err := os.Stat(path); err != nil {
		common.SysLog("geoip database is unavailable: " + err.Error())
		return nil, false
	}

	reader, err := maxminddb.Open(path)
	if err != nil {
		common.SysLog("failed to open geoip database: " + err.Error())
		return nil, false
	}

	geoIPReader = reader
	geoIPPath = path
	return geoIPReader, true
}

func firstCountryCode(codes ...string) string {
	for _, code := range codes {
		normalized := NormalizeCountryCode(code)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}
