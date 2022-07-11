package jetstreams

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	jsmapi "github.com/nats-io/jsm.go/api"
	ctrl "sigs.k8s.io/controller-runtime"

	"os"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	"github.com/edgefarm/anck/pkg/common"
	edgefarmNats "github.com/edgefarm/anck/pkg/nats"
	"github.com/nats-io/nats.go"
)

var jetstreamLog = ctrl.Log.WithName("jetstreams")

const (
	// DefaultMainDomain is the default domain for the main jetstream cluster
	DefaultMainDomain = "main"
)

// JetstreamController is a type that handle jetstreams
type JetstreamController struct {
	credsFile         string
	natsServerAddress string
}

// NewJetstreamController creates a new jetstream handler instance
func NewJetstreamController(creds string) (*JetstreamController, error) {
	natsServer, err := edgefarmNats.GetNatsServerInfos()
	if err != nil {
		return nil, err
	}
	credsFile, err := createCredsFile(creds)
	if err != nil {
		return nil, err
	}

	return &JetstreamController{
		credsFile:         credsFile,
		natsServerAddress: natsServer.Addresses.NatsAddress,
	}, nil
}

// Cleanup clears the jetstream handler
func (j *JetstreamController) Cleanup() {
	os.Remove(j.credsFile)
}

// NewJetstreamControllerWithAddress creates a new jetstream handler instance with address
func NewJetstreamControllerWithAddress(creds string, address string) (*JetstreamController, error) {
	credsFile, err := createCredsFile(creds)
	if err != nil {
		return nil, err
	}

	return &JetstreamController{
		credsFile:         credsFile,
		natsServerAddress: address,
	}, nil
}

// Exists checks if a jetstream stream exists
func (j *JetstreamController) Exists(domain string, streamName string) (bool, error) {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return false, err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return false, err
	}

	streams, err := mgr.StreamNames(&jsm.StreamNamesFilter{})
	if err != nil {
		return false, err
	}
	for _, stream := range streams {
		if stream == streamName {
			return true, nil
		}
	}
	return false, nil
}

// ListNamesNoDomain returns the names of all the jetstream streams without any domain
func (j *JetstreamController) ListNamesNoDomain() ([]string, error) {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return nil, err
	}
	defer nc.Close()

	mgr, err := jsm.New(nc)
	if err != nil {
		return nil, err
	}

	streams, err := mgr.StreamNames(&jsm.StreamNamesFilter{})
	if err != nil {
		return nil, err
	}
	return streams, nil
}

// ListNames returns the names of all the jetstream streams
func (j *JetstreamController) ListNames(domain string) ([]string, error) {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return nil, err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return nil, err
	}

	streams, err := mgr.StreamNames(&jsm.StreamNamesFilter{})
	if err != nil {
		return nil, err
	}
	return streams, nil
}

// UpdateSources updates the sources for a given jetstream stream
func (j *JetstreamController) UpdateSources(domain string, streamName string, newSources []*jsmapi.StreamSource) error {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return err
	}

	sourceStream, err := mgr.LoadStream(streamName)
	if err != nil {
		return err
	}

	// lazy deep copy
	input := sourceStream.Configuration()
	ij, err := json.Marshal(input)
	if err != nil {
		return err
	}
	var upstreamConfig api.StreamConfig
	err = json.Unmarshal(ij, &upstreamConfig)
	if err != nil {
		return err
	}

	upstreamSourcesRaw := convertSliceOfPointers(upstreamConfig.Sources)
	newSourcesRaw := convertSliceOfPointers(newSources)

	if !common.SliceEqual(upstreamSourcesRaw, newSourcesRaw) {
		upstreamConfig.Sources = newSources
		err = sourceStream.UpdateConfiguration(upstreamConfig)
		if err != nil {
			return err
		}
	}
	return nil

}

// myStreamSource is a type that represents a jetstream stream source with no pointers for deep comparison
type myStreamSource struct {
	Name          string                `json:"name"`
	OptStartSeq   uint64                `json:"opt_start_seq,omitempty"`
	OptStartTime  time.Time             `json:"opt_start_time,omitempty"`
	FilterSubject string                `json:"filter_subject,omitempty"`
	External      jsmapi.ExternalStream `json:"external,omitempty"`
}

func convertSliceOfPointers(in []*jsmapi.StreamSource) []myStreamSource {
	var out []myStreamSource
	for _, i := range in {
		out = append(out, myStreamSource{
			Name:          i.Name,
			OptStartSeq:   i.OptStartSeq,
			OptStartTime:  *i.OptStartTime,
			FilterSubject: i.FilterSubject,
			External:      *i.External,
		})
	}
	return out
}

// Get returns the jetstream stream for a given domain
func (j *JetstreamController) Get(domain string, streamName string) (*jsm.Stream, error) {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return nil, err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return nil, err
	}

	stream, err := mgr.LoadStream(streamName)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// Create creates a new jetstream stream with a given configuration for a given domain
func (j *JetstreamController) Create(domain string, network string, streamConfig networkv1alpha1.StreamSpec, subjects []networkv1alpha1.SubjectSpec) error {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return err
	}

	opts, err := createJetstreamConfig(domain, streamConfig, subjects)
	if err != nil {
		return err
	}
	opts.Name = fmt.Sprintf("%s_%s", network, streamConfig.Name)
	// jetstreamLog.Info("creating stream", "domain", domain, "name", opts.Name, "network", network)
	_, err = mgr.LoadOrNewStreamFromDefault(fmt.Sprintf("%s_%s", network, streamConfig.Name), *opts)
	if err != nil {
		return err
	}

	return nil
}

// CreateAggregate creates a new jetstream stream with a given configuration for a given domain
func (j *JetstreamController) CreateAggregate(domain string, network *networkv1alpha1.Network, streamConfig networkv1alpha1.StreamSpec, sourceDomains []string) error {

	cfg, err := createJetstreamConfig("", streamConfig, nil)
	if err != nil {
		return err
	}
	if streamConfig.Link == nil {
		return fmt.Errorf("Cannot create aggregate. No link section found in stream spec")
	}

	cfg.Sources = func() []*jsmapi.StreamSource {
		sources := []*jsmapi.StreamSource{}
		for _, sourceDomain := range sourceDomains {
			sources = append(sources, &jsmapi.StreamSource{
				Name:         fmt.Sprintf("%s_%s", network.Name, streamConfig.Link.Stream),
				OptStartSeq:  0,
				OptStartTime: &time.Time{},
				External: &jsmapi.ExternalStream{
					ApiPrefix: fmt.Sprintf("$JS.%s.API", sourceDomain),
				},
			})
		}
		return sources
	}()

	targetStreamName := fmt.Sprintf("%s_%s", network.Name, streamConfig.Name)

	cfg.Name = targetStreamName
	jetstreamLog.Info("creating aggregated stream", "domain", domain, "name", cfg.Name, "network", network.Name)

	exists, err := j.Exists(domain, targetStreamName)
	if err != nil {
		return err
	}

	if exists {
		// First check if update is working. If not, create it.
		err = j.UpdateSources(domain, targetStreamName, cfg.Sources)
		if err != nil {
			if !jsm.IsNatsError(err, 10059) {
				return err
			}
		}
	} else {
		if len(cfg.Sources) > 0 {
			// stream does not exist. Create it
			nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
			if err != nil {
				return err
			}
			defer nc.Close()

			opt := []jsm.Option{jsm.WithDomain(domain)}
			mgr, err := jsm.New(nc, opt...)
			if err != nil {
				return err
			}
			_, err = mgr.LoadOrNewStreamFromDefault(targetStreamName, *cfg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateMirror creates a new mirrored jetstream with a given configuration for a given domain
func (j *JetstreamController) CreateMirror(domain string, sourceDomain string, network *networkv1alpha1.Network, streamConfig networkv1alpha1.StreamSpec) error {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return err
	}

	opts, err := createJetstreamConfig("", streamConfig, nil)
	if err != nil {
		return err
	}
	opts.Mirror = func() *jsmapi.StreamSource {
		for _, stream := range network.Spec.Streams {
			if stream.Link != nil {
				return &jsmapi.StreamSource{
					Name:         stream.Link.Stream,
					OptStartSeq:  0,
					OptStartTime: &time.Time{},
					External: &jsmapi.ExternalStream{
						ApiPrefix: fmt.Sprintf("$JS.%s.API", sourceDomain),
					},
				}
			}

		}
		return nil
	}()
	if opts.Mirror == nil {
		return fmt.Errorf("mirror stream not found")
	}

	opts.Name = fmt.Sprintf("%s_%s", network.Name, streamConfig.Name)
	jetstreamLog.Info("creating mirrored stream", "domain", domain, "sourceDomain", sourceDomain, "name", opts.Name, "network", network)
	_, err = mgr.LoadOrNewStreamFromDefault(fmt.Sprintf("%s_%s", network.Name, streamConfig.Name), *opts)
	if err != nil {
		return err
	}

	return nil
}

// DeleteNoDomain deletes a jetstream stream with no domain
func (j *JetstreamController) DeleteNoDomain(network string, names []string) error {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
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
		jetstreamLog.Info("deleting stream", "name", stream.Name(), "network", network)
		err = stream.Delete()
		if err != nil {
			fmt.Println("error deleting stream:", err)
			errors = true
		}
	}
	if errors {
		return fmt.Errorf("error deleting streams")
	}
	return nil
}

// Delete deletes a jetstream stream for a given domain
func (j *JetstreamController) Delete(domain string, network string, names []string) error {
	nc, err := nats.Connect(j.natsServerAddress, nats.UserCredentials(j.credsFile))
	if err != nil {
		return err
	}
	defer nc.Close()

	opt := []jsm.Option{jsm.WithDomain(domain)}
	mgr, err := jsm.New(nc, opt...)
	if err != nil {
		return err
	}

	streams, err := mgr.Streams()
	if err != nil {
		return err
	}
	errors := false
	for _, stream := range streams {
		jetstreamLog.Info("deleting stream", "domain", domain, "name", stream.Name(), "network", network)
		err = stream.Delete()
		if err != nil {
			fmt.Println("error deleting stream:", err)
			errors = true
		}
	}
	if errors {
		return fmt.Errorf("error deleting streams")
	}
	return nil
}

func createJetstreamConfig(domain string, streamConfig networkv1alpha1.StreamSpec, subjects []networkv1alpha1.SubjectSpec) (*jsmapi.StreamConfig, error) {
	subjectsForStream := []string{}
	if subjects != nil {
		for _, subject := range subjects {
			if subject.Stream == streamConfig.Name {
				subjectsForStream = append(subjectsForStream, subject.Subjects...)
			}
		}
		if len(subjectsForStream) == 0 {
			return nil, fmt.Errorf("no subjects found for stream %s", streamConfig.Name)
		}
	}
	// Adding the node name as a prefix for the subjects.
	// This is needed to prevent crosstalk between leaf-nats sharing the same jetstream configuration.
	// The user has to prefix the subjects accordingly to the domain used.
	domainSubjectsForStream := []string{}
	if domain != "" {
		for _, s := range subjectsForStream {
			domainSubjectsForStream = append(domainSubjectsForStream, fmt.Sprintf("%s.%s", domain, s))
		}
	} else {
		domainSubjectsForStream = subjectsForStream
	}
	maxAge, err := parseDurationString(streamConfig.Config.MaxAge)
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
	}(streamConfig.Config.Retention)
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
	}(streamConfig.Config.Storage)
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
	}(streamConfig.Config.Discard)
	if err != nil {
		return nil, err
	}

	opts := &jsmapi.StreamConfig{
		Name:         streamConfig.Name,
		Subjects:     domainSubjectsForStream,
		Retention:    retention,
		MaxMsgsPer:   streamConfig.Config.MaxMsgsPerSubject,
		MaxMsgs:      streamConfig.Config.MaxMsgs,
		MaxBytes:     streamConfig.Config.MaxBytes,
		MaxAge:       maxAge,
		MaxMsgSize:   streamConfig.Config.MaxMsgSize,
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

// ParseCredsString parses a nats creds file and returns the jwt and nkey
func ParseCredsString(creds string) (string, string, error) {
	jwt := ""
	nkey := ""
	lines := strings.Split(creds, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i-1], "BEGIN NATS USER JWT") {
			jwt = lines[i]
			continue
		} else if strings.Contains(lines[i-1], "BEGIN USER NKEY SEED") {
			nkey = lines[i]
			continue
		}
	}
	if jwt == "" || nkey == "" {
		return "", "", fmt.Errorf("creds file does not contain both a JWT and a NKEY")
	}
	return jwt, nkey, nil
}

// parseDurationString taken from https://github.com/nats-io/natscli/blob/main/cli/util.go
func parseDurationString(dstr string) (dur time.Duration, err error) {
	dstr = strings.TrimSpace(dstr)

	if len(dstr) == 0 {
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
