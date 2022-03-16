package nats

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreds(t *testing.T) {
	assert := assert.New(t)
	creds := NewCreds("myJWT", "myNKEY")
	fmt.Println(creds)
	assert.Equal(creds, `-----BEGIN NATS USER JWT-----
myJWT
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
myNKEY
------END USER NKEY SEED------

*************************************************************
`)
}
