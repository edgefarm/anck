package dapr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDaprConfig(t *testing.T) {
	assert := assert.New(t)
	config := NewDapr("myPubsub", "myJwt", "mySeedkey")
	str, err := config.DaprConfigToYaml()
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
