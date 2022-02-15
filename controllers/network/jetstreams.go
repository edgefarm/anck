package network

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"

	"os"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	"github.com/nats-io/nats.go"
)

const (
	ngsDefaultURI = "nats://connect.ngs.global:4222"
)

// Jetstream is a type that handle jetstreams
type Jetstream struct {
	credsFile string
}

// NewJetstream creates a new jetstream handler instance
func NewJetstream(creds string) (*Jetstream, error) {
	credsFile, err := createCredsFile(creds)
	if err != nil {
		return nil, err
	}

	return &Jetstream{
		credsFile: credsFile,
	}, nil
}

// Cleanup clears the jetstream handler
func (j *Jetstream) Cleanup() {
	os.Remove(j.credsFile)
}

// Create creates a new jetstream stream with a given configuration
func (j *Jetstream) Create(streamConfig networkv1alpha1.StreamSpec) error {
	nc, err := nats.Connect(ngsDefaultURI, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	mgr, err := jsm.New(nc)
	if err != nil {
		return err
	}

	opts, err := createJetstreamConfig(streamConfig)
	if err != nil {
		return err
	}

	_, err = mgr.LoadOrNewStreamFromDefault(streamConfig.Name, *opts)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes a jetstream stream
func (j *Jetstream) Delete(names []string) error {
	nc, err := nats.Connect(ngsDefaultURI, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	mgr, err := jsm.New(nc)
	if err != nil {
		return err
	}

	streams, err := mgr.Streams()
	if err != nil {
		return err
	}
	errors := false
	for _, stream := range streams {
		if contains(names, stream.Name()) {
			err = stream.Delete()
			if err != nil {
				fmt.Println("error deleting stream:", err)
				errors = true
			}
		}
	}
	if errors {
		return fmt.Errorf("error deleting streams")
	}
	return nil
}

func createJetstreamConfig(streamConfig networkv1alpha1.StreamSpec) (*jsmapi.StreamConfig, error) {
	maxAge, err := parseDurationString(streamConfig.MaxAge)
	if err != nil {
		return nil, err
	}

	retention, err := func(policy string) (jsmapi.RetentionPolicy, error) {
		switch policy {
		case "limits":
			return jsmapi.LimitsPolicy, nil
		case "interest":
			return jsmapi.InterestPolicy, nil
		case "workqueue":
			return jsmapi.WorkQueuePolicy, nil
		}
		return jsmapi.LimitsPolicy, errors.New("invalid retention policy")
	}(streamConfig.Retention)
	if err != nil {
		return nil, err
	}

	storage, err := func(policy string) (jsmapi.StorageType, error) {
		switch policy {
		case "file":
			return jsmapi.FileStorage, nil
		case "memory":
			return jsmapi.MemoryStorage, nil
		}
		return jsmapi.MemoryStorage, errors.New("invalid storage policy")
	}(streamConfig.Storage)
	if err != nil {
		return nil, err
	}

	discard, err := func(policy string) (jsmapi.DiscardPolicy, error) {
		switch policy {
		case "old":
			return jsmapi.DiscardOld, nil
		case "new":
			return jsmapi.DiscardNew, nil
		}
		return jsmapi.DiscardOld, errors.New("invalid discard policy")
	}(streamConfig.Discard)
	if err != nil {
		return nil, err
	}

	opts := &jsmapi.StreamConfig{
		Name:         streamConfig.Name,
		Subjects:     streamConfig.Subjects,
		Retention:    retention,
		MaxMsgsPer:   streamConfig.MaxMsgsPerSubject,
		MaxMsgs:      streamConfig.MaxMsgs,
		MaxBytes:     streamConfig.MaxBytes,
		MaxAge:       maxAge,
		MaxMsgSize:   streamConfig.MaxMsgSize,
		Storage:      storage,
		Discard:      discard,
		Replicas:     1,
		NoAck:        false,
		MaxConsumers: -1,
	}
	return opts, nil
}

func createCredsFile(creds string) (string, error) {
	f, err := os.CreateTemp("", "creds")
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(creds)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

// parseDurationString taken from https://github.com/nats-io/natscli/blob/main/cli/util.go
func parseDurationString(dstr string) (dur time.Duration, err error) {
	dstr = strings.TrimSpace(dstr)

	if len(dstr) <= 0 {
		return dur, nil
	}

	ls := len(dstr)
	di := ls - 1
	unit := dstr[di:]

	switch unit {
	case "w", "W":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*7*24) * time.Hour

	case "d", "D":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24) * time.Hour
	case "M":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24*30) * time.Hour
	case "Y", "y":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24*365) * time.Hour
	case "s", "S", "m", "h", "H":
		if isUpper(dstr) {
			dstr = strings.ToLower(dstr)
		}
		dur, err = time.ParseDuration(dstr)
		if err != nil {
			return dur, err
		}

	default:
		return dur, fmt.Errorf("invalid time unit %s", unit)
	}

	return dur, nil
}

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
