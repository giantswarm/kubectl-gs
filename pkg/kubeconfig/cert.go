package kubeconfig

import (
	"os/user"
	"path"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

const (
	CertFileName = "k8s-ca.crt"
)

func GetKubeCertPath(clusterName string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", microerror.Mask(err)
	}
	kubeCertPath := path.Join(usr.HomeDir, ".kube", "certs", clusterName)

	return kubeCertPath, nil
}

func GetKubeCertFilePath(clusterName string) (string, error) {
	certPath, err := GetKubeCertPath(clusterName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	certPath = path.Join(certPath, CertFileName)

	return certPath, nil
}

func WriteCertificate(cert, clusterName string, fs afero.Fs) error {
	var err error

	var certPath string
	{
		certPath, err = GetKubeCertPath(clusterName)
		if err != nil {
			return err
		}

		err = fs.MkdirAll(certPath, 0700)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	certFilePath, err := GetKubeCertFilePath(clusterName)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, certFilePath, []byte(cert), 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
