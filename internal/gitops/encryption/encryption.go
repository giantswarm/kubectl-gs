package encryption

import (
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/giantswarm/microerror"
)

type KeyPair struct {
	Fingerprint string
	PrivateData string
	PublicData  string
}

func GenerateKeyPair(name string) (KeyPair, error) {
	// Generate a PGP key to use as private
	pgp := crypto.PGP()
	genHandle := pgp.KeyGeneration().AddUserId(name, "").New()
	prvKey, err := genHandle.GenerateKey()
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	prvArmor, err := prvKey.Armor()
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	pubArmor, err := prvKey.GetArmoredPublicKey()
	if err != nil {
		return KeyPair{}, microerror.Mask(err)
	}

	keyPair := KeyPair{
		Fingerprint: prvKey.GetFingerprint(),
		PrivateData: prvArmor,
		PublicData:  pubArmor,
	}

	return keyPair, nil
}
