// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package live

import (
	"encoding/json"
	"math/big"

	"github.com/frostymuaddib/go-ethereum-poic/common/hexutil"
)

var _ = (*supplyInfoBurnMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (s supplyInfoBurn) MarshalJSON() ([]byte, error) {
	type supplyInfoBurn struct {
		EIP1559 *hexutil.Big `json:"1559,omitempty"`
		Blob    *hexutil.Big `json:"blob,omitempty"`
		Misc    *hexutil.Big `json:"misc,omitempty"`
	}
	var enc supplyInfoBurn
	enc.EIP1559 = (*hexutil.Big)(s.EIP1559)
	enc.Blob = (*hexutil.Big)(s.Blob)
	enc.Misc = (*hexutil.Big)(s.Misc)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (s *supplyInfoBurn) UnmarshalJSON(input []byte) error {
	type supplyInfoBurn struct {
		EIP1559 *hexutil.Big `json:"1559,omitempty"`
		Blob    *hexutil.Big `json:"blob,omitempty"`
		Misc    *hexutil.Big `json:"misc,omitempty"`
	}
	var dec supplyInfoBurn
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.EIP1559 != nil {
		s.EIP1559 = (*big.Int)(dec.EIP1559)
	}
	if dec.Blob != nil {
		s.Blob = (*big.Int)(dec.Blob)
	}
	if dec.Misc != nil {
		s.Misc = (*big.Int)(dec.Misc)
	}
	return nil
}
