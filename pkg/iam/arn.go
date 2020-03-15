package iam

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

const fullArnPrefix = "arn:"

// ARNRegexp is the regex to check that the base ARN is valid,
// see http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-arns.
var ARNRegexp = regexp.MustCompile(`^arn:(\w|-)*:iam::\d+:role\/?(\w+|-|\/|\.)*$`)

// isValidBaseARN validates that the base ARN is valid.
func isValidARN(arn string) bool {
	return ARNRegexp.MatchString(arn)
}

type ARNGetter interface {
	GetARN(role string) (string, error)
}

// getARN returns the full iam role ARN.
func GetARN(role string) (string, error) {
	if isValidARN(role) {
		return role, nil
	}

	if strings.HasPrefix(strings.ToLower(role), fullArnPrefix) && !isValidARN(role) {
		return "", fmt.Errorf("%s is not a valid ARN", role)
	}

	baseArn, err := getBaseArn()
	if err != nil {
		return "", err
	}

	arn := fmt.Sprintf("%s%s", baseArn, role)
	if !isValidARN(arn) {
		return "", fmt.Errorf("%s is not a valid ARN", arn)
	}

	return arn, nil
}

func getBaseArn() (string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}
	metadata := ec2metadata.New(sess)
	if !metadata.Available() {
		return "", fmt.Errorf("EC2 Metadata is not available, are you running on EC2?")
	}
	iamInfo, err := metadata.IAMInfo()
	if err != nil {
		return "", err
	}

	arn := strings.Replace(iamInfo.InstanceProfileArn, "instance-profile", "role", 1)
	splitArn := strings.Split(arn, "/")
	if len(splitArn) < 2 {
		return "", fmt.Errorf("can't determine BaseARN")
	}

	return fmt.Sprintf("%s/", splitArn[0]), nil
}
