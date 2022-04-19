package nats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	pretty "github.com/tidwall/pretty"
)

// Config is the configuration for the NATS server
type Config struct {
	Jetstream       *Jetstream     `json:"jetstream,omitempty"`
	Authorization   *Authorization `json:"authorization,omitempty"`
	PidFile         *string        `json:"pid_file,omitempty"`
	HTTP            int            `json:"http"`
	Leafnodes       *Leafnodes     `json:"leafnodes,omitempty"`
	Operator        *string        `json:"operator,omitempty"`
	SystemAccount   *string        `json:"system_account,omitempty"`
	Resolver        *Resolver      `json:"resolver,omitempty"`
	ResolverPreload interface{}    `json:"resolver_preload,omitempty"`
}

// User contains the user information
type User struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Jetstream is the jetstream configuration for the NATS server
type Jetstream struct {
	StoreDir string `json:"store_dir"`
	Domain   string `json:"domain"`
}

// Authorization is the authorization configuration
type Authorization struct {
	Users []User `json:"users,omitempty"`
}

// Remotes is the remote configuration part
type Remotes struct {
	// URL is the remote url, e.g. tls://connect.ngs.global:7422 or nats://localhost:4222
	URL string `json:"url"`
	// Path to creds file
	Credentials string `json:"credentials"`
	// Account public key
	Account string `json:"account"`
	// DenyImports is a list of subjects to deny imports for
	DenyImports []string `json:"deny_imports,omitempty"`
	// DenyExports is a list of subjects to deny exports for
	DenyExports []string `json:"deny_exports,omitempty"`
}

// Leafnodes is the leafnode configuration part
type Leafnodes struct {
	Remotes []Remotes `json:"remotes"`
}

// Resolver is the resolver configuration part
type Resolver struct {
	Type        string  `json:"type"`
	Dir         string  `json:"dir"`
	AllowDelete *bool   `json:"allow_delete,omitempty"`
	Interval    *string `json:"interval,omitempty"`
	TTL         *string `json:"ttl,omitempty"`
	Timeout     string  `json:"timeout"`
}

// Option is a type that represents a Config option
type Option func(*Config)

// WithPidFile sets a leafnode remote
func WithPidFile(pidFile string) Option {
	return func(c *Config) {
		c.PidFile = &pidFile
	}
}

// WithJetstream enabled jetstream and sets the store dir and an domain
func WithJetstream(storageDir string, domain string) Option {
	return func(c *Config) {
		if c.Jetstream == nil {
			c.Jetstream = &Jetstream{}
		}
		c.Jetstream.StoreDir = storageDir
		c.Jetstream.Domain = domain
	}
}

// WithRemote sets a leafnode remote
func WithRemote(url string, credentials string, accountPublicKey string, denyImports []string, denyExports []string) Option {
	return func(c *Config) {
		err := c.AddRemote(url, credentials, accountPublicKey, denyImports, denyExports)
		if err != nil {
			panic(err)
		}
	}
}

// WithNGSRemote sets a leafnode remote to NGS
func WithNGSRemote(credentials string, accountPublicKey string, denyImports []string, denyExports []string) Option {
	return WithRemote("tls://connect.ngs.global:7422", credentials, accountPublicKey, denyImports, denyExports)
}

// WithFullResolver sets a full resolver (used for nats account servers)
func WithFullResolver(operatorJWT string, sysAccountPubKey string, sysAccountJWT string, jwtStoragePath string) Option {
	return func(c *Config) {
		c.Operator = func() *string { s := operatorJWT; return &s }()
		c.SystemAccount = func() *string { s := sysAccountPubKey; return &s }()
		c.Resolver = func() *Resolver {
			return &Resolver{
				Type:        "full",
				Dir:         jwtStoragePath,
				AllowDelete: func() *bool { b := false; return &b }(),
				Interval:    func() *string { s := "2m"; return &s }(),
				Timeout:     "1.9s",
			}
		}()
		// put in resolver_preload in the raw json, because the key cannot be
		// named in the go struct
		raw := make(map[string]interface{})
		err := json.Unmarshal([]byte("{}"), &raw)
		if err != nil {
			panic(err)
		}
		raw["resolver_preload"] = func() map[string]interface{} {
			return map[string]interface{}{
				sysAccountPubKey: sysAccountJWT,
			}
		}()
		j, err := json.Marshal(raw)
		if err != nil {
			panic(err)
		}
		filled := &Config{}
		err = json.Unmarshal(j, filled)
		if err != nil {
			panic(err)
		}
		c.ResolverPreload = filled.ResolverPreload
	}
}

// WithCacheResolver sets a cached resolver (used for leaf nats servers)
func WithCacheResolver(operatorJWT string, sysAccountPubKey string, sysAccountJWT string, jwtStoragePath string) Option {
	return func(c *Config) {
		c.Operator = func() *string { s := operatorJWT; return &s }()
		c.SystemAccount = func() *string { s := sysAccountPubKey; return &s }()
		c.Resolver = func() *Resolver {
			return &Resolver{
				Type:    "cache",
				Dir:     jwtStoragePath,
				TTL:     func() *string { s := "1h"; return &s }(),
				Timeout: "1.9s",
			}
		}()
		// put in resolver_preload in the raw json, because the key cannot be
		// named in the go struct
		raw := make(map[string]interface{})
		err := json.Unmarshal([]byte("{}"), &raw)
		if err != nil {
			panic(err)
		}
		raw["resolver_preload"] = func() map[string]interface{} {
			return map[string]interface{}{
				sysAccountPubKey: sysAccountJWT,
			}
		}()
		j, err := json.Marshal(raw)
		if err != nil {
			panic(err)
		}
		filled := &Config{}
		err = json.Unmarshal(j, filled)
		if err != nil {
			panic(err)
		}
		c.ResolverPreload = filled.ResolverPreload
	}
}

// WithAdminUser sets an admin user
func WithAdminUser(user string, password string) Option {
	return func(c *Config) {
		if c.Authorization == nil {
			c.Authorization = &Authorization{}
		}
		if c.Authorization.Users == nil {
			c.Authorization.Users = make([]User, 0)
		}
		c.Authorization.Users = append(c.Authorization.Users, User{
			User:     user,
			Password: password,
		})
	}
}

// NewConfig creates a new Config instance
func NewConfig(opts ...Option) *Config {
	config := &Config{
		HTTP: 8222,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// ToJSON converts a Config instance to Json
func (c *Config) ToJSON() (string, error) {
	// cannot use json.Marshal(*c) here because it escapes '>' and '<'
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(*c)
	if err != nil {
		return "", err
	}

	return string(pretty.Pretty(bf.Bytes())), nil
}

// LoadFromFile loads a config ftom a JSON file
func LoadFromFile(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadFromJSON(string(bytes))
}

// LoadFromJSON loads a config from a JSON string
func LoadFromJSON(j string) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal([]byte(j), config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// AddNGSRemote adds a NGS remote to the config
func (c *Config) AddNGSRemote(credentials string, accountPublicKey string, denyImports []string, denyExports []string) error {
	return c.AddRemote("tls://connect.ngs.global:7422", credentials, accountPublicKey, denyImports, denyExports)
}

// AddRemote adds a remote to the configs
func (c *Config) AddRemote(url string, credentials string, accountPublicKey string, denyImports []string, denyExports []string) error {
	if c.Leafnodes == nil {
		c.Leafnodes = &Leafnodes{}
	}
	if c.Leafnodes.Remotes == nil {
		c.Leafnodes.Remotes = make([]Remotes, 0)
	}
	c.Leafnodes.Remotes = append(c.Leafnodes.Remotes, Remotes{
		URL:         url,
		Credentials: credentials,
		Account:     accountPublicKey,
		DenyImports: denyImports,
		DenyExports: denyExports,
	})
	return nil
}

// RemoveRemoteByAccountPubKey removes a remote by account public key
func (c *Config) RemoveRemoteByAccountPubKey(accountPublicKey string) error {
	if c.Leafnodes == nil {
		return nil
	}
	if c.Leafnodes.Remotes == nil {
		return nil
	}
	found := false
	for i, r := range c.Leafnodes.Remotes {
		if r.Account == accountPublicKey {
			c.Leafnodes.Remotes = append(c.Leafnodes.Remotes[:i], c.Leafnodes.Remotes[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("remote with account public key %s not found", accountPublicKey)
	}
	return nil
}

// RemoveRemoteByCredsfile removes a remote by credentials files path
func (c *Config) RemoveRemoteByCredsfile(path string) error {
	if c.Leafnodes == nil {
		return nil
	}
	if c.Leafnodes.Remotes == nil {
		return nil
	}
	found := false
	for i, r := range c.Leafnodes.Remotes {
		if strings.Contains(r.Credentials, path) {
			c.Leafnodes.Remotes = append(c.Leafnodes.Remotes[:i], c.Leafnodes.Remotes[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("remote with creds file %s not found", path)
	}
	return nil
}
