package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	v1beta1 "k8s.io/kms/apis/v1beta1"
	v2 "k8s.io/kms/apis/v2"
)

type fakePlugin struct {
	encryptResponse []byte
	decryptResponse []byte
	keyVersion      string
	encryptErr      error
	decryptErr      error
	keyVersionErr   error
}

func (f *fakePlugin) Encrypt(ctx context.Context, data []byte) ([]byte, string, error) {
	return f.encryptResponse, f.keyVersion, f.encryptErr
}

func (f *fakePlugin) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	return f.decryptResponse, f.decryptErr
}

func (f *fakePlugin) GetKeyVersion(ctx context.Context) (string, error) {
	return f.keyVersion, f.keyVersionErr
}

func TestKMSv1UsesPluginInterface(t *testing.T) {
	t.Parallel()

	kms := NewPluginV1(&fakePlugin{
		encryptResponse: []byte("cipher"),
		decryptResponse: []byte("plain"),
		keyVersion:      "1",
	})

	enc, err := kms.Encrypt(context.Background(), &v1beta1.EncryptRequest{Plain: []byte("plain")})
	require.NoError(t, err)
	require.Equal(t, []byte("cipher"), enc.GetCipher())

	dec, err := kms.Decrypt(context.Background(), &v1beta1.DecryptRequest{Cipher: []byte("cipher")})
	require.NoError(t, err)
	require.Equal(t, []byte("plain"), dec.GetPlain())
}

func TestKMSv2StatusReflectsHealthFailures(t *testing.T) {
	t.Parallel()

	kms := NewPluginV2(&fakePlugin{
		encryptResponse: []byte("cipher"),
		decryptResponse: []byte("wrong"),
		keyVersion:      "7",
	})

	resp, err := kms.Status(context.Background(), &v2.StatusRequest{})
	require.NoError(t, err)
	require.Equal(t, "v2", resp.GetVersion())
	require.Equal(t, "err", resp.GetHealthz())
	require.Equal(t, "7", resp.GetKeyId())
}

func TestKMSv2StatusReturnsKeyVersionErrors(t *testing.T) {
	t.Parallel()

	kms := NewPluginV2(&fakePlugin{
		keyVersionErr: errors.New("vault unavailable"),
	})

	_, err := kms.Status(context.Background(), &v2.StatusRequest{})
	require.EqualError(t, err, "vault unavailable")
}
