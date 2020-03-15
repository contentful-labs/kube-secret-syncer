package secretsmanager

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	lru "github.com/hashicorp/golang-lru"
)

func TestFetchCurrentSecret(t *testing.T) {
	type Want struct {
		resp  *secretsmanager.GetSecretValueOutput
		found bool
	}
	type Have struct {
		poller        *Poller
		secretID      string
		secretVersion string
		lruElements   map[string]map[string]secretsmanager.GetSecretValueOutput
	}
	for _, test := range []struct {
		name string
		have Have
		want Want
	}{
		{
			name: "when the cache is dirty",
			have: Have{
				poller: &Poller{
					PolledSecrets: Secrets{
						"cf/secret/test": PolledSecretMeta{
							CurrentVersionID: "present",
							UpdatedAt:        time.Now().AddDate(0, 0, -2),
						},
					},
				},
				secretID: "cf/secret/test",
				lruElements: map[string]map[string]secretsmanager.GetSecretValueOutput{
					"cf/secret/test": {
						"": {
							VersionId: _s("past"),
						},
					},
				},
			},
			want: Want{
				resp:  nil,
				found: false,
			},
		},
		{
			name: "when the cache is valid",
			have: Have{
				poller: &Poller{
					PolledSecrets: Secrets{
						"cf/secret/test": PolledSecretMeta{
							CurrentVersionID: "present",
							UpdatedAt:        time.Now().AddDate(0, 0, -2),
						},
					},
				},
				secretID: "cf/secret/test",
				lruElements: map[string]map[string]secretsmanager.GetSecretValueOutput{
					"cf/secret/test": {
						"": {
							VersionId: _s("present"),
						},
					},
				},
			},
			want: Want{
				resp: &secretsmanager.GetSecretValueOutput{
					VersionId: _s("present"),
				},
				found: true,
			},
		},
		{
			name: "when the polledcache is empty",
			have: Have{
				poller: &Poller{
					PolledSecrets: Secrets{},
				},
				secretID: "cf/secret/test",
				lruElements: map[string]map[string]secretsmanager.GetSecretValueOutput{
					"cf/secret/test": {
						"": {
							VersionId: _s("present"),
						},
					},
				},
			},
			want: Want{
				resp:  nil,
				found: false,
			},
		},
	} {
		test.have.poller.cachedSecretValuesByRole, _ = lru.New2Q(10)
		for k, v := range test.have.lruElements {
			test.have.poller.cachedSecretValuesByRole.Add(k, v)
		}

		gotResp, gotFound := test.have.poller.fetchCurrentSecretCache(&test.have.secretID, "")
		if !reflect.DeepEqual(gotResp, test.want.resp) {
			t.Errorf("resp doesn't match %s. Wanted %v, got %v", test.name, test.want.resp, gotResp)
		}
		if gotFound != test.want.found {
			t.Errorf("found doesn't match %s. Wanted %v, got %v", test.name, test.want.found, gotFound)
		}
	}
}
