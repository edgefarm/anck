package dapr

import "gopkg.in/yaml.v2"

const (
	natsURLValue string = "nats://nats.nats:4222"
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

// NewDapr creates a new Config instance
func NewDapr(name, jwt, seedkey string) *Config {
	return &Config{
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
					Value: natsURLValue,
				},
				{
					Name:  "jwt",
					Value: jwt,
				},
				{
					Name:  "seedkey",
					Value: seedkey,
				},
			},
		},
	}
}

// DaprConfigToYaml converts a DaprConfig instance to YAML
func (d *Config) DaprConfigToYaml() (string, error) {
	yaml, err := yaml.Marshal(*d)
	if err != nil {
		return "", err
	}

	return string(yaml), nil
}
