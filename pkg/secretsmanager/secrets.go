package secretsmanager

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
)

func FilterByTagKey(secrets Secrets, tagKey string) Secrets {
	filteredSecrets := Secrets{}
	for secretName, secretMeta := range secrets {
		for tagName, _ := range secretMeta.Tags {
			if tagName == tagKey {
				filteredSecrets[secretName] = secretMeta
			}
		}
	}

	return filteredSecrets
}

// GetCurrentSecret Returns the secret value for `secretId` with stage `AWSCURRENT`
// TODO add a test to ensure this is mocked well including the error
func (p *Poller) GetSecret(secretID *string, IAMRole string) (string, string, error) {
	if secretValueOut, ok := p.fetchCurrentSecretCache(secretID, IAMRole); ok {
		return *secretValueOut.SecretString, *secretValueOut.VersionId, nil
	}

	smClient, err := p.getSMClient(IAMRole)
	if err != nil {
		return "", "", err
	}

	// Not in cache, or new versionID found
	secretValueOut, err := smClient.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId:     secretID,
		VersionStage: aws.String("AWSCURRENT"),
	})
	if err != nil {
		return "", "", errors.WithMessagef(err, "can't find AWSCURRENT version for secretID %s", *secretID)
	}

	if cachedElem, ok := p.cachedSecretValuesByRole.Get(*secretID); !ok {
		cachedElem := map[string]secretsmanager.GetSecretValueOutput{
			IAMRole: *secretValueOut,
		}
		p.cachedSecretValuesByRole.Add(*secretID, cachedElem)
	} else {
		cachedElem.(map[string]secretsmanager.GetSecretValueOutput)[IAMRole] = *secretValueOut
	}

	return *secretValueOut.SecretString, *secretValueOut.VersionId, nil
}

func (p *Poller) fetchCurrentSecretCache(secretID *string, role string) (*secretsmanager.GetSecretValueOutput, bool) {
	if cachedElem, ok := p.cachedSecretValuesByRole.Get(*secretID); ok {
		//old secretValueOut := cachedElem.(map[string]*secretsmanager.GetSecretValueOutput)
		secretValuesByRole := cachedElem.(map[string]secretsmanager.GetSecretValueOutput)
		if secretValueOut, ok := secretValuesByRole[role]; ok {
			polledSecretMeta, found := p.PolledSecrets[*secretID]
			if found && polledSecretMeta.CurrentVersionID == *secretValueOut.VersionId {
				return &secretValueOut, found
			}
		}
	}

	return nil, false
}
