/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/contentful-labs/kube-secret-syncer/pkg/k8snamespace"

	"github.com/aws/aws-sdk-go/aws"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	secretsv1 "github.com/contentful-labs/kube-secret-syncer/api/v1"
	"github.com/contentful-labs/kube-secret-syncer/controllers"
	"github.com/contentful-labs/kube-secret-syncer/pkg/iam"
	"github.com/contentful-labs/kube-secret-syncer/pkg/rolevalidator"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = secretsv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type SMSVCFactory struct {
	session       *session.Session
	arns          iam.ARNGetter
	SMSVC         secretsmanageriface.SecretsManagerAPI            // Main, default SM service - used when no IAM Role is specified in the secret
	AssumedSMSVCs map[string]secretsmanageriface.SecretsManagerAPI // SM Service for each IAM Role
}

func getDurationFromEnv(envVar string, defaultDuration time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(envVar)
	if ok {
		if value == "" {
			return defaultDuration, nil
		}

		valueInt, err := strconv.Atoi(value)
		if err == nil {
			interval := time.Second * time.Duration(valueInt)
			return interval, nil
		}
		return 0 * time.Second, fmt.Errorf("%s invalid: %s", envVar, value)
	}
	return defaultDuration, nil
}

func (s SMSVCFactory) getSMSVC(iamRole string) (secretsmanageriface.SecretsManagerAPI, error) {
	var smsvc secretsmanageriface.SecretsManagerAPI
	var err error

	// No iamRole specified, we use the default service
	if iamRole == "" {
		return s.SMSVC, nil
	}

	// ensure specified iamRole is an ARN
	iamGetARN, err := s.arns.GetARN(iamRole)
	if err != nil {
		return nil, err
	}

	var ok bool
	smsvc, ok = s.AssumedSMSVCs[iamGetARN]
	if !ok {
		creds := stscreds.NewCredentials(s.session, iamGetARN)
		smsvc = secretsmanager.New(s.session, &aws.Config{Credentials: creds})
		s.AssumedSMSVCs[iamGetARN] = smsvc
	}

	return smsvc, nil
}

func newSMSVCFactory(sess *session.Session, arnGetter iam.ARNGetter) *SMSVCFactory {
	return &SMSVCFactory{
		session:       sess,
		arns:          arnGetter,
		SMSVC:         secretsmanager.New(sess),
		AssumedSMSVCs: map[string]secretsmanageriface.SecretsManagerAPI{},
	}
}

func realMain() int {
	metricsAddr := os.Getenv("METRICS_LISTEN")
	if metricsAddr == "" {
		metricsAddr = ":8080"
	}

	annotationName := os.Getenv("NS_ANNOTATION")
	if annotationName == "" {
		annotationName = "iam.amazonaws.com/allowed-roles"
	}

	syncPeriod, err := getDurationFromEnv("SYNC_INTERVAL_SEC", 120*time.Second)
	if err != nil {
		setupLog.Error(err, "failed parsing SYNC_INTERVAL_SEC: should be an integer")
		return 1
	}

	pollInterval, err := getDurationFromEnv("POLL_INTERVAL_SEC", 300*time.Second)
	if err != nil {
		setupLog.Error(err, "failed parsing POLL_INTERVAL_SEC: should be an integer")
		return 1
	}

	logCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	stackTraceLevel := uzap.NewAtomicLevelAt(zapcore.PanicLevel)
	ctrl.SetLogger(zap.New(zap.Encoder(zapcore.NewJSONEncoder(logCfg)), zap.StacktraceLevel(&stackTraceLevel)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     true,
		LeaderElectionID:   "5a48bfe8.contentful.com",
		SyncPeriod:         &syncPeriod,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return 1
	}

	ctx := context.Background()

	Retry5Cfg := request.WithRetryer(aws.NewConfig(), awsclient.DefaultRetryer{NumMaxRetries: 5})
	arnClient := iam.NewARNClientWithCache(iam.GetARN)
	smsvcfactory := newSMSVCFactory(session.Must(session.NewSession(Retry5Cfg)), arnClient)

	nsCache, err := k8snamespace.NewWatcher(ctx)
	if err != nil {
		setupLog.Error(err, "unable to start namespace watcher")
		return 1
	}

	roleValidator := rolevalidator.NewRoleValidator(arnClient, nsCache, annotationName)

	r := &controllers.SyncedSecretReconciler{
		Client:        mgr.GetClient(),
		Ctx:           ctx,
		Log:           ctrl.Log.WithName("controllers").WithName("SyncedSecret"),
		Sess:          session.New(Retry5Cfg),
		GetSMClient:   smsvcfactory.getSMSVC,
		RoleValidator: roleValidator,
		PollInterval:  pollInterval,
	}

	// Introduce artificial startup delay so that all controllers do not start
	// polling SecretsManager at the same time
	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	initialDelayS := time.Duration(r1.Intn(int(pollInterval / time.Second))) * time.Second
	time.Sleep(initialDelayS)

	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SyncedSecret")
		return 1
	}
	defer r.Quit()

	// +kubebuilder:scaffold:builder
	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return 1
	}

	return 0
}

func main() {
	// Call realMain so that defers work properly, since os.Exit won't
	// call defers.
	os.Exit(realMain())
}
