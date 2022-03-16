package nats

import (
	"context"
	"fmt"

	"github.com/edgefarm/anck/pkg/common"
	"github.com/hsson/once"
	"google.golang.org/grpc"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
)

var (
	instance *SysAccount
)

// SysAccount is the system account
type SysAccount struct {
	OperatorJWT      string
	SysAccountJWT    string
	SysAccountCreds  string
	SysAccountPubKey string
}

// GetSysAccount returns the system account
func GetSysAccount() (*SysAccount, error) {
	o := once.Error{}
	err := o.Do(func() error {

		cc, err := grpc.Dial(common.AnckcredentialsServiceURL, grpc.WithInsecure())
		if err != nil {
			return err
		}
		defer cc.Close()
		grpcclient := anckcredentials.NewConfigServiceClient(cc)

		res, err := grpcclient.SysAccount(context.Background(), &anckcredentials.SysAccountRequest{})
		if err != nil {
			return err
		}

		instance = &SysAccount{
			OperatorJWT:      res.OperatorJWT,
			SysAccountJWT:    res.SysJWT,
			SysAccountCreds:  res.SysCreds,
			SysAccountPubKey: res.SysPublicKey,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if instance == nil {
		return nil, fmt.Errorf("SysAccount was not initialized properly")
	}

	return instance, nil
}

// GetOperatorJWT returns the operator JWT
func (s *SysAccount) GetOperatorJWT() string {
	return s.OperatorJWT
}

// GetSysAccountJWT returns the system account JWT
func (s *SysAccount) GetSysAccountJWT() string {
	return s.SysAccountJWT
}

// GetSysAccountCreds returns the system account credentials
func (s *SysAccount) GetSysAccountCreds() string {
	return s.SysAccountCreds
}

// GetSysAccountPubKey returns the system account public key
func (s *SysAccount) GetSysAccountPubKey() string {
	return s.SysAccountPubKey
}
