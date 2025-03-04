package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewPool is the constructor of Pool
func NewPool(name string) Pool {
	return Pool{
		Name:    name,
		Rewards: []Pool_Reward{},
	}
}

// Validate returns an error if the pool is not valid; nil otherwise
func (m Pool) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name not set for pool")
	}

	validatorSeen := make(map[string]bool)
	for _, reward := range m.Rewards {
		validatorAddr := reward.Validator.String()
		if validatorSeen[validatorAddr] {
			return fmt.Errorf("duplicate validator %s found in pool %s", validatorAddr, m.Name)
		}

		if err := sdk.VerifyAddressFormat(reward.Validator); err != nil {
			return fmt.Errorf("invalid validator %s found in pool %s", validatorAddr, m.Name)
		}

		if reward.Coins == nil || reward.Coins.Empty() {
			return fmt.Errorf("empty rewards found for validator %s in pool %s", validatorAddr, m.Name)
		}

		if err := reward.Coins.Validate(); err != nil {
			return sdkerrors.Wrapf(err, "invalid rewards for validator %s found in pool %s", validatorAddr, m.Name)
		}

		validatorSeen[validatorAddr] = true
	}

	return nil
}
