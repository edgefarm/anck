package nats

import (
	"fmt"
	"testing"

	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/assert"
)

func TestNatsConfig(t *testing.T) {
	assert := assert.New(t)
	config := NewConfig()
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{"http": 8222}`)
}

func TestNatsFullConfig(t *testing.T) {
	assert := assert.New(t)
	opts := []Option{}
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"))
	opts = append(opts, WithFullResolver("operatorJWT", "sysPubKey", "sysJWT", "/jwt"))
	config := NewConfig(opts...)
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			}
		  ]
		},
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
		  "type": "full",
		  "dir": "/jwt",
		  "allow_delete": false,
		  "interval": "2m",
		  "timeout": "1.9s"
		},
		"resolver_preload": {
		  "sysPubKey": "sysJWT"
		}
	  }`)
}

func TestNatsCacheResolverConfig(t *testing.T) {
	assert := assert.New(t)
	opts := []Option{}
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"))
	opts = append(opts, WithCacheResolver("operatorJWT", "sysPubKey", "sysJWT", "/jwt"))
	config := NewConfig(opts...)
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			}
		  ]
		},
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
		  "type": "cache",
		  "dir": "/jwt",
		  "ttl": "1h",
		  "timeout": "1.9s"
		},
		"resolver_preload": {
		  "sysPubKey": "sysJWT"
		}
	  }`)
}

func TestNatsWithRemotes(t *testing.T) {
	assert := assert.New(t)
	config := NewConfig(WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"), WithRemote("nats://localohst:4222", "/my/bar.creds", "BAR"))
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/bar.creds",
			  "account": "BAR"
			}
		  ]
		}
	  }`)
}

func TestNatsWithRemotesAndPidFile(t *testing.T) {
	assert := assert.New(t)
	opts := []Option{}

	opts = append(opts, WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"))
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/bar.creds", "BAR"))
	opts = append(opts, WithPidFile("/tmp/foo.pid"))

	config := NewConfig(opts...)
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"pid_file": "/tmp/foo.pid",
		"http": 8222,
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/bar.creds",
			  "account": "BAR"
			}
		  ]
		}
	  }`)
}

func TestNatsLoadFromString(t *testing.T) {
	assert := assert.New(t)
	config, err := LoadFromJSON(
		`{
	"http": 8222,
	"leafnodes": {
	  "remotes": [
		{
		  "url": "nats://localohst:4222",
		  "credentials": "/my/foo.creds",
		  "account": "FOO"
		}
	  ]
	},
	"operator": "operatorJWT",
	"system_account": "sysPubKey",
	"resolver": {
	  "type": "cache",
	  "dir": "/jwt",
	  "ttl": "1h",
	  "timeout": "1.9s"
	},
	"resolver_preload": {
	  "sysPubKey": "sysJWT"
	}
  }`)
	assert.Nil(err)
	assert.Equal(config.HTTP, int(8222))
	assert.Equal(*config.Operator, "operatorJWT")
	assert.Equal(*config.SystemAccount, "sysPubKey")
	assert.Equal(config.Resolver.Type, "cache")
	assert.Equal(config.Resolver.Dir, "/jwt")
	assert.Equal(*config.Resolver.TTL, "1h")
	assert.Equal(config.Resolver.Timeout, "1.9s")
	assert.Equal(config.ResolverPreload.(map[string]interface{})["sysPubKey"], "sysJWT")

}

func TestNatsAddRemote(t *testing.T) {
	assert := assert.New(t)
	opts := []Option{}
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"))
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/bar.creds", "BAR"))
	opts = append(opts, WithFullResolver("operatorJWT", "sysPubKey", "sysJWT", "/jwt"))
	config := NewConfig(opts...)

	err := config.AddRemote("nats://localohst:4222", "/my/baz.creds", "BAZ")
	assert.Nil(err)
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
		  "type": "full",
		  "dir": "/jwt",
		  "allow_delete": false,
		  "interval": "2m",
		  "timeout": "1.9s"
		},
		"resolver_preload": {
		  "sysPubKey": "sysJWT"
		},
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/bar.creds",
			  "account": "BAR"
			},
			{
				"url": "nats://localohst:4222",
				"credentials": "/my/baz.creds",
				"account": "BAZ"
			  }
		  ]
		}
	  }`)

	err = config.RemoveRemoteByAccountPubKey("BAR")
	assert.Nil(err)
	str, err = config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
		  "type": "full",
		  "dir": "/jwt",
		  "allow_delete": false,
		  "interval": "2m",
		  "timeout": "1.9s"
		},
		"resolver_preload": {
		  "sysPubKey": "sysJWT"
		},
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
				"url": "nats://localohst:4222",
				"credentials": "/my/baz.creds",
				"account": "BAZ"
			  }
		  ]
		}
	  }`)

	err = config.RemoveRemoteByAccountPubKey("NOTEXISTENT")
	assert.NotNil(err)
	str, err = config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
		  "type": "full",
		  "dir": "/jwt",
		  "allow_delete": false,
		  "interval": "2m",
		  "timeout": "1.9s"
		},
		"resolver_preload": {
		  "sysPubKey": "sysJWT"
		},
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
				"url": "nats://localohst:4222",
				"credentials": "/my/baz.creds",
				"account": "BAZ"
			  }
		  ]
		}
	  }`)

	err = config.RemoveRemoteByCredsfile("foo.creds")
	assert.Nil(err)
	str, err = config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"http": 8222,
		"operator": "operatorJWT",
		"system_account": "sysPubKey",
		"resolver": {
			"type": "full",
			"dir": "/jwt",
			"allow_delete": false,
			"interval": "2m",
			"timeout": "1.9s"
			},
		"resolver_preload": {
		"sysPubKey": "sysJWT"
		},
		  "leafnodes": {
			"remotes": [
			  {
				  "url": "nats://localohst:4222",
				  "credentials": "/my/baz.creds",
				  "account": "BAZ"
				}
			]
		  }
		}`)

	err = config.RemoveRemoteByCredsfile("missing.creds")
	assert.NotNil(err)
	str, err = config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
			"http": 8222,
			"operator": "operatorJWT",
			"system_account": "sysPubKey",
			"resolver": {
				"type": "full",
				"dir": "/jwt",
				"allow_delete": false,
				"interval": "2m",
				"timeout": "1.9s"
				},
			"resolver_preload": {
			"sysPubKey": "sysJWT"
			},
			  "leafnodes": {
				"remotes": [
				  {
					  "url": "nats://localohst:4222",
					  "credentials": "/my/baz.creds",
					  "account": "BAZ"
					}
				]
			  }
			}`)
}

func TestNatsWithRemotesPidFileAndAdminUser(t *testing.T) {
	assert := assert.New(t)
	opts := []Option{}

	opts = append(opts, WithRemote("nats://localohst:4222", "/my/foo.creds", "FOO"))
	opts = append(opts, WithRemote("nats://localohst:4222", "/my/bar.creds", "BAR"))
	opts = append(opts, WithAdminUser("admin", "Unicorn5322"))
	opts = append(opts, WithPidFile("/tmp/foo.pid"))

	config := NewConfig(opts...)
	str, err := config.ToJSON()
	fmt.Println(str)
	assert.Nil(err)
	jsonassert.New(t).Assertf(str, `{
		"pid_file": "/tmp/foo.pid",
		"authorization": {
			"users": [
			  {
				"user": "admin",
				"password": "Unicorn5322"
			  }
			]
		},
		"http": 8222,
		"leafnodes": {
		  "remotes": [
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/foo.creds",
			  "account": "FOO"
			},
			{
			  "url": "nats://localohst:4222",
			  "credentials": "/my/bar.creds",
			  "account": "BAR"
			}
		  ]
		}
	  }`)
}
