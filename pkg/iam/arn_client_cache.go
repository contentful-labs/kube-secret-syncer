package iam

type ArnClientCached struct {
	arnCache  map[string]string
	arnGetter func(string) (string, error)
}

func NewARNClientWithCache(getter func(string) (string, error)) *ArnClientCached {
	return &ArnClientCached{
		arnCache:  map[string]string{},
		arnGetter: getter,
	}
}

func (ag *ArnClientCached) GetARN(role string) (string, error) {
	var err error

	arn, ok := ag.arnCache[role]
	if !ok {
		arn, err = ag.arnGetter(role)
		if err != nil {
			return "", err
		}
		ag.arnCache[role] = arn
	}
	return arn, err
}
