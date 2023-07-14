package license

import (
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"github.com/gemfast/server/internal/config"
	"github.com/keygen-sh/keygen-go/v2"
	"github.com/rs/zerolog/log"
)

type License struct {
	Account     string
	Product     string
	LicenseKey  string
	Fingerprint string
	Validated   bool
	Machine     *keygen.Machine
}

func NewLicense(cfg *config.Config) (*License, error) {
	var l *License
	fingerprint, err := machineid.ProtectedID(keygen.Product)
	if err != nil {
		return nil, err
	}
	l = &License{
		Account:     "5590bc22-b3de-4e34-a27a-7cc07c3ba683",
		Product:     "2c4f54ab-c7a0-4f74-bfbd-9f4973c21121",
		LicenseKey:  cfg.LicenseKey,
		Fingerprint: fingerprint,
	}

	err = l.validateLicenseKey()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *License) configureKeygen() {
	keygen.Account = "5590bc22-b3de-4e34-a27a-7cc07c3ba683"
	keygen.Product = "2c4f54ab-c7a0-4f74-bfbd-9f4973c21121"
	keygen.LicenseKey = l.LicenseKey
}

func (l *License) validateLicenseKey() error {
	if l.LicenseKey == "" {
		log.Info().Msg("no license_key supplied in gemfast.hcl")
		log.Info().Msg("consider purchasing a license from https://gemfast.io")
		return nil
	}
	l.configureKeygen()
	// Validate the license for the current fingerprint
	license, err := keygen.Validate(l.Fingerprint)
	switch {
	case err == keygen.ErrLicenseNotActivated:
		_, err := license.Activate(l.Fingerprint)
		switch {
		case err == keygen.ErrMachineLimitExceeded:
			log.Error().Err(err).Msg("gemfast machine limit has been exceeded")
			return fmt.Errorf("gemfast machine limit has been exceeded")
		case err != nil:
			log.Error().Err(err).Msg("gemfast license is expired")
			return fmt.Errorf("gemfast machine activation failed")
		}
	case err == keygen.ErrLicenseExpired:
		log.Error().Err(err).Msg("gemfast license is expired")
		return fmt.Errorf("gemfast license is expired")
	case err != nil:
		log.Error().Err(err).Msg("gemfast license is invalid")
		return fmt.Errorf("gemfast license is invalid")
	}

	log.Info().Msg("gemfast license is valid")
	l.Validated = true
	m, err := license.Machine(l.Fingerprint)
	if err != nil {
		log.Error().Err(err).Msg("unable to get machine from fingerprint")
	}
	l.Machine = m
	return nil
}
