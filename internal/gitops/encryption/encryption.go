package encryption

import (
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/giantswarm/microerror"
)

type KeyPair struct {
	Fingerprint string
	PrivateData string
	PublicData  string
}

func GenerateKeyPair(name string) (KeyPair, error) {
	encKeys, err := crypto.GenerateKey(name, "", "x25519", 0)
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	prvArmor, err := encKeys.Armor()
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	pubArmor, err := encKeys.GetArmoredPublicKey()
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	keyPair := KeyPair{
		Fingerprint: encKeys.GetFingerprint(),
		PrivateData: prvArmor,
		PublicData:  pubArmor,
	}

	return keyPair, nil
}
