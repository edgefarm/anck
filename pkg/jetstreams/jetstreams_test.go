package jetstreams

import (
	"testing"
	"time"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/stretchr/testify/assert"
)

const (
	credsFile = `-----BEGIN NATS USER JWT-----
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
LASDKFN89Q4ORAKJF94THJAOGJHAQ04YUJEOGHJPSGJK908JWPEGJWE8TW
------END USER NKEY SEED------

*************************************************************
`
)

func TestCreateCredsfile(t *testing.T) {
	assert := assert.New(t)
	creds, err := createCredsFile(credsFile)
	assert.Nil(err)
	assert.NotEmpty(creds)
	assert.Contains(creds, "/tmp/")
}

func TestCreateJetstreamConfigsStorageFile(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}
	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.FileStorage)
	assert.Equal(c.Retention, jsmapi.InterestPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigsStorageMemory(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "memory",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.MemoryStorage)
	assert.Equal(c.Retention, jsmapi.InterestPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigInvalidStorage(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "invalid",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.NotNil(err)
	assert.Nil(c)
}

func TestCreateJetstreamConfigsRetentionLimits(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "memory",
			Retention:         "limits",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.MemoryStorage)
	assert.Equal(c.Retention, jsmapi.LimitsPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigsRetentionInterest(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "memory",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.MemoryStorage)
	assert.Equal(c.Retention, jsmapi.InterestPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigsRetentionWorkqueue(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "memory",
			Retention:         "workqueue",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.MemoryStorage)
	assert.Equal(c.Retention, jsmapi.WorkQueuePolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigInvalidRetention(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "invalid",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.NotNil(err)
	assert.Nil(c)
}

func TestCreateJetstreamConfigDiscardNew(t *testing.T) {
	assert := assert.New(t)
	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "new",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.FileStorage)
	assert.Equal(c.Retention, jsmapi.InterestPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardNew)
}

func TestCreateJetstreamConfigDiscardOld(t *testing.T) {
	assert := assert.New(t)

	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "old",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.Nil(err)
	assert.NotNil(c)
	assert.Equal(c.Name, "myName")
	assert.Equal(c.Subjects, []string{"mySubject1", "mySubject2"})
	assert.Equal(c.Storage, jsmapi.FileStorage)
	assert.Equal(c.Retention, jsmapi.InterestPolicy)
	assert.Equal(c.MaxMsgsPer, int64(100))
	assert.Equal(c.MaxMsgs, int64(1000))
	assert.Equal(c.MaxBytes, int64(1000000))
	assert.Equal(c.MaxAge, time.Hour*24*234)
	assert.Equal(c.MaxMsgSize, int32(1000000))
	assert.Equal(c.Discard, jsmapi.DiscardOld)
}

func TestCreateJetstreamConfigInvalidDiscard(t *testing.T) {
	assert := assert.New(t)

	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "234d",
			MaxMsgSize:        1000000,
			Discard:           "invalid",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.NotNil(err)
	assert.Nil(c)
}

func TestCreateJetstreamConfigInvalidMaxAge(t *testing.T) {
	assert := assert.New(t)

	config := networkv1alpha1.StreamSpec{
		Name:     "myName",
		Location: "node",
		Config: networkv1alpha1.StreamConfigSpec{
			Storage:           "file",
			Retention:         "interest",
			MaxMsgsPerSubject: 100,
			MaxMsgs:           1000,
			MaxBytes:          1000000,
			MaxAge:            "invalid",
			MaxMsgSize:        1000000,
			Discard:           "invalid",
		},
	}

	c, err := createJetstreamConfig("", config, []networkv1alpha1.SubjectSpec{
		{
			Name:     "mystreams",
			Subjects: []string{"mySubject1", "mySubject2"},
			Stream:   "myName",
		},
	})
	assert.NotNil(err)
	assert.Nil(c)
}

func TestParseDurationString(t *testing.T) {
	assert := assert.New(t)

	dur, err := parseDurationString("1h")
	assert.Nil(err)
	assert.Equal(dur, time.Hour)

	dur, err = parseDurationString("1H")
	assert.Nil(err)
	assert.Equal(dur, time.Hour)

	dur, err = parseDurationString("1m")
	assert.Nil(err)
	assert.Equal(dur, time.Minute)

	dur, err = parseDurationString("30s")
	assert.Nil(err)
	assert.Equal(dur, time.Second*30)

	dur, err = parseDurationString("30S")
	assert.Nil(err)
	assert.Equal(dur, time.Second*30)

	dur, err = parseDurationString("2d")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*2)

	dur, err = parseDurationString("2D")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*2)

	dur, err = parseDurationString("2M")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*30*2)

	dur, err = parseDurationString("1y")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*365)

	dur, err = parseDurationString("1Y")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*365)

	dur, err = parseDurationString("1w")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*7)

	dur, err = parseDurationString("1W")
	assert.Nil(err)
	assert.Equal(dur, time.Hour*24*7)
}

func TestParseDurationStringErrors(t *testing.T) {
	assert := assert.New(t)

	dur, err := parseDurationString("")
	assert.Nil(err)
	assert.Equal(dur, time.Duration(0))

	// second
	_, err = parseDurationString("s")
	assert.NotNil(err)

	_, err = parseDurationString("as")
	assert.NotNil(err)

	_, err = parseDurationString(".s")
	assert.NotNil(err)

	_, err = parseDurationString("S")
	assert.NotNil(err)

	_, err = parseDurationString("/S")
	assert.NotNil(err)

	_, err = parseDurationString(".S")
	assert.NotNil(err)

	// minute
	_, err = parseDurationString("m")
	assert.NotNil(err)

	_, err = parseDurationString("asm")
	assert.NotNil(err)

	_, err = parseDurationString(".m")
	assert.NotNil(err)

	_, err = parseDurationString("M")
	assert.NotNil(err)

	_, err = parseDurationString("/M")
	assert.NotNil(err)

	_, err = parseDurationString(".M")
	assert.NotNil(err)

	// hour
	_, err = parseDurationString("h")
	assert.NotNil(err)

	_, err = parseDurationString("ah")
	assert.NotNil(err)

	_, err = parseDurationString(".h")
	assert.NotNil(err)

	_, err = parseDurationString("H")
	assert.NotNil(err)

	_, err = parseDurationString("/H")
	assert.NotNil(err)

	_, err = parseDurationString(".H")
	assert.NotNil(err)

	// day
	_, err = parseDurationString("d")
	assert.NotNil(err)

	_, err = parseDurationString("ad")
	assert.NotNil(err)

	_, err = parseDurationString(".d")
	assert.NotNil(err)

	_, err = parseDurationString("D")
	assert.NotNil(err)

	_, err = parseDurationString("/D")
	assert.NotNil(err)

	_, err = parseDurationString(".D")
	assert.NotNil(err)

	// week
	_, err = parseDurationString("w")
	assert.NotNil(err)

	_, err = parseDurationString("aw")
	assert.NotNil(err)

	_, err = parseDurationString(".w")
	assert.NotNil(err)

	_, err = parseDurationString("W")
	assert.NotNil(err)

	_, err = parseDurationString("/W")
	assert.NotNil(err)

	_, err = parseDurationString(".W")
	assert.NotNil(err)

	// month
	_, err = parseDurationString("M")
	assert.NotNil(err)

	_, err = parseDurationString("aM")
	assert.NotNil(err)

	_, err = parseDurationString(".M")
	assert.NotNil(err)

	// year
	_, err = parseDurationString("y")
	assert.NotNil(err)

	_, err = parseDurationString("ay")
	assert.NotNil(err)

	_, err = parseDurationString(".y")
	assert.NotNil(err)

	_, err = parseDurationString("Y")
	assert.NotNil(err)

	_, err = parseDurationString("/Y")
	assert.NotNil(err)

	_, err = parseDurationString(".Y")
	assert.NotNil(err)

	// unknown unit
	_, err = parseDurationString("a")
	assert.NotNil(err)
}
