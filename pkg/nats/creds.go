package nats

import (
	"strings"
)

// NewCreds creates a new nats creds file
func NewCreds(jwt string, nkey string) string {
	template := `-----BEGIN NATS USER JWT-----
ENTER-JWT-HERE
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
ENTER-NKEY-HERE
------END USER NKEY SEED------

*************************************************************
`
	ret := strings.ReplaceAll(template, "ENTER-JWT-HERE", jwt)
	ret = strings.ReplaceAll(ret, "ENTER-NKEY-HERE", nkey)

	return ret
}
