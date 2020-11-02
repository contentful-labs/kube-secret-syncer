package secretsmanager

import (
	"github.com/aws/aws-sdk-go/aws"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
)

type SecretGetter interface {
	GetSecret(secretID *string) (string, string, error)
}
type Secrets map[string]PolledSecretMeta

type Poller struct {
	PolledSecrets Secrets
	getSMClient   func(string) (secretsmanageriface.SecretsManagerAPI, error)

	smLastPolledOn           time.Time
	cachedSecretValuesByRole *lru.TwoQueueCache
	wg                       sync.WaitGroup
	errs                     chan<- error
	quit                     chan bool
}

// SecretMeta meta information of a polled secret
type PolledSecretMeta struct {
	Tags             map[string]string
	CurrentVersionID string
	UpdatedAt        time.Time
}

// New creates a new poller, will send polling or other non critical errors through the errs channel
func New(interval time.Duration, errs chan error, getSMClient func(string) (secretsmanageriface.SecretsManagerAPI, error)) (*Poller, error) {
	p := &Poller{
		errs:        errs,
		getSMClient: getSMClient,
		quit:        make(chan bool),
	}
	var err error

	// init a lru cache that can hold 10000 items (arbit value for now)
	// this doesn't init the size to value set here, but is only used to figure if eviction is required or not
	p.cachedSecretValuesByRole, err = lru.New2Q(10000)
	if err != nil {
		return nil, err
	}

	// poll in sync the first time to ensure that we have a populated cache before reconciler kicks in
	p.PolledSecrets, err = p.fetchSecrets()
	if err != nil {
		return nil, err
	}

	go func() {
		p.wg.Add(1)
		ticker := time.NewTicker(interval)
		p.poll(ticker)
		ticker.Stop()
		p.wg.Done()
	}()

	return p, nil
}

func (p *Poller) Stop() {
	p.quit <- true
	p.wg.Wait()
}

// poller polls secrets manager at `tick` defined intervals, caches it locally,
func (p *Poller) poll(ticker *time.Ticker) {
	for {
		select {
		case _ = <-ticker.C:
			polledSecrets, err := p.fetchSecrets()
			if err != nil {
				p.errs <- errors.WithMessagef(err, "failed polling secrets")
			} else {
				p.PolledSecrets = polledSecrets
			}

		case <-p.quit:
			close(p.errs)
			return
		}
	}
}

func (p *Poller) fetchSecrets() (Secrets, error) {
	fetchedSecrets := make(Secrets)

	allSecrets := []*secretsmanager.SecretListEntry{}
	input := &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int64(100),
	}

	smClient, err := p.getSMClient("")
	if err != nil {
		return nil, err
	}

	err = smClient.ListSecretsPages(input, func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		allSecrets = append(allSecrets, page.SecretList...)
		return !lastPage
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return nil, errors.WithMessagef(aerr, "failed listing secrets, error code: %s", aerr.Code())
		}
		return nil, errors.WithMessagef(err, "failed listing secrets")
	}

	for _, secret := range allSecrets {
		if secret.DeletedDate != nil {
			continue
		}

		versionID, err := getCurrentVersion(secret.SecretVersionsToStages)
		if err != nil {
			continue
		}

		secretTags := map[string]string{}
		for _, t := range secret.Tags {
			secretTags[*t.Key] = *t.Value
		}

		fetchedSecrets[*secret.Name] = PolledSecretMeta{
			Tags:             secretTags,
			CurrentVersionID: versionID,
			UpdatedAt:        *secret.LastChangedDate,
		}
	}

	p.smLastPolledOn = time.Now().UTC()
	return fetchedSecrets, nil
}

// getCurrentVersion finds the versionid with AWSCURRENT
func getCurrentVersion(secretVersionToStages map[string][]*string) (string, error) {
	for uuid, stages := range secretVersionToStages {
		for _, stage := range stages {
			if *stage == "AWSCURRENT" {
				return uuid, nil
			}
		}
	}
	return "", errors.New("version with stage AWSCURRENT not found")
}
