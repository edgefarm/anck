package dapr

import "gopkg.in/yaml.v2"

const (
	defaultNatsURL string = "nats://connect.ngs.global:4222"
)

// Config is a type that represents a Config instance
type Config struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata is a type that represents a Config metadata
type Metadata struct {
	Name string `yaml:"name"`
}

// SpecMetadata is a type that represents a Config spec metadata
type SpecMetadata struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Spec is a type that represents a Config spec
type Spec struct {
	Type         string         `yaml:"type"`
	Version      string         `yaml:"version"`
	SpecMetadata []SpecMetadata `yaml:"metadata"`
}

// Option is a type that represents a Config option
type Option func(*Config)

// WithCreds sets the credentials for the DaprConfig
func WithCreds(jwt string, nkey string) Option {
	return func(c *Config) {
		c.Spec.SpecMetadata = append(c.Spec.SpecMetadata, SpecMetadata{
			Name:  "jwt",
			Value: jwt,
		})
		c.Spec.SpecMetadata = append(c.Spec.SpecMetadata, SpecMetadata{
			Name:  "seedKey",
			Value: nkey,
		})
	}
}

// WithNatsURL sets the NATS URL for the DaprConfig
func WithNatsURL(url string) Option {
	return func(c *Config) {
		urlIndex := 0
		for i, m := range c.Spec.SpecMetadata {
			if m.Name == "natsURL" {
				urlIndex = i
				break
			}
		}
		c.Spec.SpecMetadata[urlIndex].Value = url
	}
}

// NewDapr creates a new Config instance
func NewDapr(name string, opts ...Option) *Config {
	// Default config
	config := &Config{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
		Metadata: Metadata{
			Name: name,
		},
		Spec: Spec{
			Type:    "pubsub.jetstream",
			Version: "v1",
			SpecMetadata: []SpecMetadata{
				{
					Name:  "natsURL",
					Value: defaultNatsURL,
				},
			},
		},
	}

	// Loop through each option
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// ToYaml converts a DaprConfig instance to YAML
func (d *Config) ToYaml() (string, error) {
	yaml, err := yaml.Marshal(*d)
	if err != nil {
		return "", err
	}

	return string(yaml), nil
}
