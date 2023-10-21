package providers

type Provider interface {
	RetrieveSecrets(envVars []string) ([]string, error)

	//Future implementation.
	//RenewSecret()
}
