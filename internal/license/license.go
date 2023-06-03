package license

import (
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"github.com/gemfast/server/internal/config"
	"github.com/keygen-sh/keygen-go/v2"
	"github.com/rs/zerolog/log"
)

func ValidateLicenseKey() error {
	if config.Cfg.LicenseKey == "" {
		log.Info().Msg("no license key supplied in GEMFAST_LICENSE_KEY variable")
		log.Info().Msg("consider purchasing a license from https://gemfast.io")
		log.Info().Msg("services will be started in trial mode")
		config.TrialMode()
		return nil
	}
	keygen.Account = "5590bc22-b3de-4e34-a27a-7cc07c3ba683"
	keygen.Product = "2c4f54ab-c7a0-4f74-bfbd-9f4973c21121"
	keygen.LicenseKey = config.Cfg.LicenseKey
	fingerprint, err := machineid.ProtectedID(keygen.Product)
	if err != nil {
		return err
	}

	// Validate the license for the current fingerprint
	license, err := keygen.Validate(fingerprint)
	switch {
	case err == keygen.ErrLicenseNotActivated:
		_, err := license.Activate(fingerprint)
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
	config.Cfg.TrialMode = false
	return nil
}
