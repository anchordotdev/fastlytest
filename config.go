//go:build !wasip1 || nofastlyhostcalls

package fastlytest

type Config struct {
	Authors         []string `toml:"authors,omitempty"`
	Description     string   `toml:"description,omitempty"`
	Language        string   `toml:"language,omitempty"`
	ManifestVersion int      `toml:"manifest_version,omitempty"`
	Name            string   `toml:"name,omitempty"`

	LocalServer `toml:"local_server,omitempty"`
}

type Setup struct {
}

type LocalServer struct {
	Backends     map[string]Backend     `toml:"backends,omitempty"`
	ConfigStores map[string]ConfigStore `toml:"config_stores,omitempty"`
	KVStores     map[string][]StoreItem `toml:"kv_stores,omitempty"`
	SecretStores map[string][]StoreItem `toml:"secret_stores,omitempty"`
	Geolocation  `toml:"geolocation,omitempty"`
}

type Backend struct {
	URL string `toml:"url,omitempty"`
}

type ConfigStore struct {
	File   string `toml:"file,omitempty"`
	Format string `toml:"format,omitempty"`

	Contents map[string]string `toml:"contents,omitempty"`
}

type StoreItem struct {
	Key  string `toml:"key,omitempty"`
	File string `toml:"file,omitempty"`
	Data string `toml:"data,omitempty"`
}

type Geolocation struct {
	Format    string                 `toml:"format,omitempty"`
	File      string                 `toml:"file,omitempty"`
	Addresses map[string]GeoVariable `toml:"addresses,omitempty"`
}

type GeoVariable struct {
	ASName           string  `toml:"as_name,omitempty"`
	ASNumber         int     `toml:"as_number,omitempty"`
	AreaCode         int     `toml:"area_code,omitempty"`
	City             string  `toml:"city,omitempty"`
	ConnSpeed        string  `toml:"conn_speed,omitempty"`
	ConnType         string  `toml:"conn_type,omitempty"`
	Continent        string  `toml:"continent,omitempty"`
	countryCode      string  `toml:"country_code,omitempty"`
	CountryCode3     string  `toml:"country_code3,omitempty"`
	CountryName      string  `toml:"country_name,omitempty"`
	Latitude         float64 `toml:"latitude,omitempty"`
	Longitude        float64 `toml:"longitude,omitempty"`
	MetroCode        int     `toml:"metro_code,omitempty"`
	PostalCode       string  `toml:"postal_code,omitempty"`
	ProxyDescription string  `toml:"proxy_description,omitempty"`
	ProxyType        string  `toml:"proxy_type,omitempty"`
	Region           string  `toml:"region,omitempty"`
	UTCOffset        int     `toml:"utc_offset,omitempty"`
}
