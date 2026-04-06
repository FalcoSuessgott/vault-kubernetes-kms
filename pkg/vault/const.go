package vault

const (
	appRoleAuthLoginPath = "auth/%s/login"

	userPassAuthLoginPath = "auth/%s/login/%s"

	encryptDataPath = "%s/encrypt/%s"
	decryptDataPath = "%s/decrypt/%s"

	mountEnginePath = "sys/mounts/%s"
	transitKeyPath  = "%s/keys/%s"
)
