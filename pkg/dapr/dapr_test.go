package dapr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDaprConfig(t *testing.T) {
	assert := assert.New(t)
	config := NewDapr("myPubsub")
	str, err := config.ToYaml()
	assert.Nil(err)
	assert.Equal(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: myPubsub
spec:
  type: pubsub.jetstream
  version: v1
  metadata:
  - name: natsURL
    value: nats://nats.nats:4222
`, str)
}

func TestCreateDaprConfigWithCreds(t *testing.T) {
	assert := assert.New(t)
	config := NewDapr("myPubsub", WithCreds("myJwt", "mySeedkey"))
	str, err := config.ToYaml()
	assert.Nil(err)
	assert.Equal(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: myPubsub
spec:
  type: pubsub.jetstream
  version: v1
  metadata:
  - name: natsURL
    value: nats://nats.nats:4222
  - name: jwt
    value: myJwt
  - name: seedkey
    value: mySeedkey
`, str)
}

func TestCreateDaprConfigWithCredsAndNatsURL(t *testing.T) {
	assert := assert.New(t)
	config := NewDapr("myPubsub", WithCreds("myJwt", "mySeedkey"), WithNatsURL("nats://mynats.example.com:4222"))
	str, err := config.ToYaml()
	assert.Nil(err)
	assert.Equal(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: myPubsub
spec:
  type: pubsub.jetstream
  version: v1
  metadata:
  - name: natsURL
    value: nats://mynats.example.com:4222
  - name: jwt
    value: myJwt
  - name: seedkey
    value: mySeedkey
`, str)
}

func TestCreateDaprConfigWithNatsURL(t *testing.T) {
	assert := assert.New(t)
	config := NewDapr("myPubsub", WithNatsURL("nats://mynats.example.com:4222"))
	str, err := config.ToYaml()
	assert.Nil(err)
	assert.Equal(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: myPubsub
spec:
  type: pubsub.jetstream
  version: v1
  metadata:
  - name: natsURL
    value: nats://mynats.example.com:4222
`, str)
}
