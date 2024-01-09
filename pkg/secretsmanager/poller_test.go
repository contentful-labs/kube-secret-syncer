package secretsmanager

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	testr "github.com/go-logr/logr/testr"
)

func TestGetCurrentVersion(t *testing.T) {
	currV := "AWSCURRENT"
	prevV := "AWSPREVIOUS"

	testCases := []struct {
		have map[string][]*string
		want string
		err  error
	}{
		{
			have: map[string][]*string{
				"currentuuid": {&currV},
			},
			want: "currentuuid",
			err:  nil,
		},
		{
			have: map[string][]*string{
				"prevuuid": {&prevV},
			},
			want: "",
			err:  errors.New("version with stage AWSCURRENT not found"),
		},
		{
			have: map[string][]*string{
				"oldversionuuid": {&prevV},
				"newversionuuid": {&currV},
			},
			want: "newversionuuid",
			err:  nil,
		},
	}

	for _, test := range testCases {
		versionID, err := getCurrentVersion(test.have)
		if test.err != nil {
			if err.Error() != test.err.Error() {
				t.Errorf("error: wanted %s got %s", test.err, err)
			}
		}
		if test.want != versionID {
			t.Errorf("versionId: wanted %s got %s", test.want, versionID)
		}
	}
}

type mockSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI
	Resp secretsmanager.ListSecretsOutput
}

func (m *mockSecretsManagerClient) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	fn(&m.Resp, true)
	return nil
}

func _s(A string) *string {
	return &A
}

func _t(A time.Time) *time.Time {
	return &A
}

func TestFetchSecret(t *testing.T) {
	var now = time.Now()

	for _, test := range []struct {
		name string
		have mockSecretsManagerClient
		want Secrets
	}{
		{
			name: "test 1",
			have: mockSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: []*secretsmanager.SecretListEntry{
						{
							Name:            _s("random/aws/secret002"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"002": {
									_s("AWSCURRENT"),
								},
							},
						}, {
							Name:            _s("random/aws/secret003"),
							LastChangedDate: _t(now.AddDate(0, 0, -3)),
							SecretVersionsToStages: map[string][]*string{
								"005": {
									_s("AWSCURRENT"),
								},
								"003": {
									_s("AWSPREVIOUS"),
								},
							},
						},
					},
				},
			},
			want: Secrets{
				"random/aws/secret002": PolledSecretMeta{
					CurrentVersionID: "002",
					UpdatedAt:        now.AddDate(0, 0, -2),
					Tags:             map[string]string{},
				},
				"random/aws/secret003": PolledSecretMeta{
					CurrentVersionID: "005",
					UpdatedAt:        now.AddDate(0, 0, -3),
					Tags:             map[string]string{},
				},
			},
		}, {
			name: "test 2",
			have: mockSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: []*secretsmanager.SecretListEntry{
						{
							Name:            _s("random/aws/secret"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid": {
									_s("AWSCURRENT"),
								},
							},
						}, {
							Name:            _s("random/aws/secretnocurrent"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid_hidden": {
									_s("AWSPREVIOUS"),
								},
							},
						},
					},
				},
			},
			want: Secrets{
				"random/aws/secret": PolledSecretMeta{
					CurrentVersionID: "randomuuid",
					UpdatedAt:        now.AddDate(0, 0, -2),
					Tags:             map[string]string{},
				},
			},
		}, {
			name: "test 3",
			have: mockSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: []*secretsmanager.SecretListEntry{
						{
							Name:            _s("random/aws/secret"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid": {
									_s("AWSSTAGE"),
								},
							},
						},
					},
				},
			},
			want: Secrets{},
		},
	} {
		p := Poller{
			getSMClient: func(string) (secretsmanageriface.SecretsManagerAPI, error) {
				return &test.have, nil
			},
			Log: testr.New(t),
		}
		got, err := p.fetchSecrets()
		if err != nil {
			t.Errorf("test %s returned error %s", test.name, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("test %s, wanted %s got %s", test.name, test.want, got)
		}
	}
}

type mockFailingSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI
	Resp secretsmanager.ListSecretsOutput
}

func (m *mockFailingSecretsManagerClient) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	return fmt.Errorf("an error occured")
}

func TestFetchSecretError(t *testing.T) {
	for _, test := range []struct {
		name string
		have mockFailingSecretsManagerClient
		want Secrets
	}{
		{
			name: "fetchSecrets should return an error when listing secret in AWS fails",
			have: mockFailingSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: nil,
				},
			},
		},
	} {
		p := Poller{
			getSMClient: func(string) (secretsmanageriface.SecretsManagerAPI, error) {
				return &test.have, nil
			},
		}
		got, err := p.fetchSecrets()
		if err == nil {
			t.Errorf("test %s should have returned an error, did not", test.name)
		}
		if got != nil {
			t.Errorf("test %s, wanted %s got %s", test.name, test.want, got)
		}
	}
}

type mockWorkingThenFailingSecretsManagerClient struct {
	count int
	secretsmanageriface.SecretsManagerAPI
	Resp secretsmanager.ListSecretsOutput
}

func (m *mockWorkingThenFailingSecretsManagerClient) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	if m.count < 2 {
		m.count = m.count + 1
		fn(&m.Resp, true)
		return nil
	} else {
		return fmt.Errorf("an error occured")
	}
}

func TestPoll(t *testing.T) {
	now := time.Now()

	for _, test := range []struct {
		name string
		have mockWorkingThenFailingSecretsManagerClient
		want Secrets
	}{
		{
			name: "test 2",
			have: mockWorkingThenFailingSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: []*secretsmanager.SecretListEntry{
						{
							Name:            _s("random/aws/secret"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid": {
									_s("AWSCURRENT"),
								},
							},
						}, {
							Name:            _s("random/aws/secretnocurrent"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid_hidden": {
									_s("AWSPREVIOUS"),
								},
							},
						},
					},
				},
			},
			want: Secrets{
				"random/aws/secret": PolledSecretMeta{
					CurrentVersionID: "randomuuid",
					UpdatedAt:        now.AddDate(0, 0, -2),
					Tags:             map[string]string{},
				},
			},
		},
		{
			name: "test 3",
			have: mockWorkingThenFailingSecretsManagerClient{
				Resp: secretsmanager.ListSecretsOutput{
					SecretList: []*secretsmanager.SecretListEntry{
						{
							Name:            _s("random/aws/secret"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid": {
									_s("AWSCURRENT"),
								},
							},
						}, {
							Name:            _s("random/aws/deletedsecret"),
							LastChangedDate: _t(now.AddDate(0, 0, -2)),
							DeletedDate:     _t(now.AddDate(0, 0, -1)),
							SecretVersionsToStages: map[string][]*string{
								"randomuuid": {
									_s("AWSCURRENT"),
								},
							},
						},
					},
				},
			},
			want: Secrets{
				"random/aws/secret": PolledSecretMeta{
					CurrentVersionID: "randomuuid",
					UpdatedAt:        now.AddDate(0, 0, -2),
					Tags:             map[string]string{},
				},
			},
		},
	} {
		errs := make(chan error)
		p := Poller{
			getSMClient: func(string) (secretsmanageriface.SecretsManagerAPI, error) {
				return &test.have, nil
			},
			errs: errs,
			quit: make(chan bool),
			Log:  testr.NewWithOptions(t, testr.Options{Verbosity: 0}),
		}

		go func() {
			p.wg.Add(1)
			ticker := time.NewTicker(time.Duration(100) * time.Millisecond)
			p.poll(ticker)
			ticker.Stop()
			p.wg.Done()
		}()

		nErrs := 0
		go func() {
			for range errs {
				nErrs = nErrs + 1
			}
		}()

		time.Sleep(500 * time.Millisecond)

		p.quit <- true
		p.wg.Wait()

		if nErrs == 0 {
			t.Errorf("there was no error listing secret - there should have been")
		}

		if len(p.PolledSecrets) != len(test.want) {
			t.Errorf("failing to list secrets seems to have removed the list of PolledSecrets")
		}
	}
}
