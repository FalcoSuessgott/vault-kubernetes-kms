package vault

const (
	appRoleAuthLoginPath = "auth/%s/login"

	userPassAuthLoginPath = "auth/%s/login/%s" //nolint:gosec

	certAuthLoginPath = "auth/%s/login"

	jwtAuthLoginPath = "auth/%s/login"

	encryptDataPath = "%s/encrypt/%s"
	decryptDataPath = "%s/decrypt/%s"

	mountEnginePath = "sys/mounts/%s"
	transitKeyPath  = "%s/keys/%s"
)
