package vault

const (
	authLoginPath = "auth/%s/login"

	encryptDataPath = "%s/encrypt/%s"
	decryptDataPath = "%s/decrypt/%s"

	mountEnginePath = "sys/mounts/%s"
	transitKeyPath  = "%s/keys/%s"
)
