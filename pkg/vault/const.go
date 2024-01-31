package vault

const (
	k8sLoginPath = "auth/%s/login"

	encryptDataPath = "%s/encrypt/%s"
	decryptDataPath = "%s/decrypt/%s"

	mountEnginePath = "sys/mounts/%s"
	transitKeyPath  = "%s/keys/%s"

	tokenRefreshIntervall = 3600

	//nolint: gosec
	serviceAccountTokenLocation = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)
