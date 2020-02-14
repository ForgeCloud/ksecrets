package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"cloud.google.com/go/secretmanager/apiv1beta1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awssecretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	secretspb "google.golang.org/genproto/googleapis/cloud/secrets/v1beta1"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

// plugin is data used to generate a secret
type plugin struct {
	helpers               *resmap.PluginHelpers
	Metadata              *types.ObjectMeta
	Source                string   `json:"source,omitempty" yaml:"source,omitempty"`
	GCPProjectID          string   `json:"gcpProjectID,omitempty" yaml:"gcpProjectID,omitempty"`
	AWSRegion             string   `json:"awsRegion,omitempty" yaml:"awsRegion,omitempty"`
	DisableNameSuffixHash bool     `json:"disableNameSuffixHash,omitempty" yaml:"disableNameSuffixHash,omitempty"`
	Type                  string   `json:"type,omitempty" yaml:"type,omitempty"`
	Behavior              string   `json:"behavior,omitempty" yaml:"behavior,omitempty"`
	Keys                  []string `json:"keys,omitempty" yaml:"keys,omitempty"`
}

// KustomizePlugin is exported for Kustomize
var KustomizePlugin plugin

// Config prepares the plugin struct's data
func (p *plugin) Config(helpers *resmap.PluginHelpers, content []byte) error {
	p.helpers = helpers
	return yaml.Unmarshal(content, p)
}

// Generate is called by Kustomize
func (p *plugin) Generate() (resmap.ResMap, error) {
	secrets, err := p.getSecrets()
	if err != nil {
		return nil, err
	}
	return p.makeResMap(secrets)
}

// getSecrets gets the secret data out of the chosen secrets manager
func (p *plugin) getSecrets() (secrets map[string]string, err error) {
	switch p.Source {
	case "GCP":
		secrets, err = p.getGCPSecrets()
		if err != nil {
			return nil, err
		}
	case "AWS":
		secrets, err = p.getAWSSecrets()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unsupported source '%s', use 'GCP' or 'AWS'", p.Source)
	}

	var yamlMap []byte
	err = yaml.Unmarshal(yamlMap, &secrets)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal failure")
	}

	return
}

func (p *plugin) getGCPSecrets() (secrets map[string]string, err error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	secrets = make(map[string]string)
	for _, key := range p.Keys {
		sanitizedKeyName := sanitizeKeyName(key)
		name := fmt.Sprintf("projects/%s/secrets/%s_%s/versions/latest", p.GCPProjectID, p.Metadata.Name, sanitizedKeyName)
		request := &secretspb.AccessSecretVersionRequest{Name: name}
		secret, err := client.AccessSecretVersion(ctx, request)
		if err != nil {
			return nil, errors.Wrapf(err, "trouble retrieving secret: %s", name)
		}
		secrets[key] = string(secret.GetPayload().GetData())
	}

	return
}

func (p *plugin) getAWSSecrets() (secrets map[string]string, err error) {
	service := awssecretsmanager.New(session.New(&aws.Config{Region: aws.String(p.AWSRegion)}))

	secrets = make(map[string]string)
	for _, key := range p.Keys {
		sanitizedKeyName := sanitizeKeyName(key)
		name := fmt.Sprintf("%s_%s", p.Metadata.Name, sanitizedKeyName)
		request := &awssecretsmanager.GetSecretValueInput{SecretId: aws.String(name)}
		result, err := service.GetSecretValue(request)
		if err != nil {
			return nil, err
		}
		secret := ""
		if result.SecretString != nil {
			secret = *result.SecretString
		} else {
			decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
			length, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
			if err != nil {
				return nil, err
			}
			secret = string(decodedBinarySecretBytes[:length])
		}
		secrets[key] = secret
	}

	return
}

// makeResMap turns a YAML string map into a ResMap, adding chosen options
func (p *plugin) makeResMap(secrets map[string]string) (resmap.ResMap, error) {
	args := types.SecretArgs{
		GeneratorArgs: types.GeneratorArgs{
			Name:      p.Metadata.Name,
			Namespace: p.Metadata.Namespace,
			Behavior:  p.Behavior,
		},
		Type: p.Type,
	}
	for key, value := range secrets {
		args.LiteralSources = append(args.LiteralSources, key+"="+value)
	}
	options := &types.GeneratorOptions{
		DisableNameSuffixHash: p.DisableNameSuffixHash,
	}
	return p.helpers.ResmapFactory().FromSecretArgs(
		kv.NewLoader(p.helpers.Loader(), p.helpers.Validator()), options, args)
}

func sanitizeKeyName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, ".", "_"), "/", "_")
}
