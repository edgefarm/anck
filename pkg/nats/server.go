package nats

import (
	"context"
	"fmt"
	"time"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	api "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	common "github.com/edgefarm/anck/pkg/common"
	grpcClient "github.com/edgefarm/anck/pkg/grpc"
	"github.com/hsson/once"
)

var (
	natsServerInfosInstance *api.ServerInformationResponse
)

// GetNatsServerInfos returns the nats server information
func GetNatsServerInfos() (*api.ServerInformationResponse, error) {
	o := once.Error{}
	err := o.Do(func() error {
		cc, err := grpcClient.Dial(common.AnckcredentialsServiceURL, time.Second*10, time.Second*1)
		if err != nil {
			return err
		}
		defer cc.Close()
		grpcclient := anckcredentials.NewConfigServiceClient(cc)

		natsServerInfosInstance, err = grpcclient.ServerInformation(context.Background(), &api.ServerInformationRequest{})
		if err != nil {
			return err
		}
		return nil

	})
	if err != nil {
		return nil, err
	}
	if natsServerInfosInstance == nil {
		return nil, fmt.Errorf("NatsServerInfos was not initialized properly")
	}

	return natsServerInfosInstance, nil
}
