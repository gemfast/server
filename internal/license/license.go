package license

import (
	"fmt"
	"github.com/denisbrodbeck/machineid"
  "github.com/keygen-sh/keygen-go/v2"
  "github.com/gemfast/server/internal/config"
)

func ValidateLicenseKey() (error) {
	keygen.Account = "5590bc22-b3de-4e34-a27a-7cc07c3ba683"
	keygen.Product = "2c4f54ab-c7a0-4f74-bfbd-9f4973c21121"
	keygen.LicenseKey = config.Env.GemfastLicenseKey
	fingerprint, err := machineid.ProtectedID(keygen.Product)
	// fmt.Println(machineid.ID())
  if err != nil {
    panic(err)
  }

  // Validate the license for the current fingerprint
  license, err := keygen.Validate(fingerprint)
  switch {
  case err == keygen.ErrLicenseNotActivated:
    // Activate the current fingerprint
    _, err := license.Activate(fingerprint)
    switch {
    case err == keygen.ErrMachineLimitExceeded:
      return fmt.Errorf("machine limit has been exceeded!")
    case err != nil:
      return fmt.Errorf("machine activation failed!")
    }
  case err == keygen.ErrLicenseExpired:
    return fmt.Errorf("license is expired!")
  case err != nil:
  	fmt.Println(err)
    return fmt.Errorf("license is invalid!")
  }

  fmt.Println("License is activated!")
  return nil
}