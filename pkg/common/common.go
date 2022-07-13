package common

import "fmt"

const (
	// AnckEdgefarmSysSecretName is the name of the secret that contains all data for the edgefarm-sys account for nats
	AnckEdgefarmSysSecretName = "edgefarm-sys"
	// AnckcredentialsServiceName is the name of the service that provides the credentials
	AnckcredentialsServiceName = "anck-credentials"
	// AnckcredentialsServicePort is the port on which the anck-credentials service is running
	AnckcredentialsServicePort = 6000
	// AnckNamespace is the namespace in which everything anck related is placed in
	AnckNamespace = "anck"

	// AnckEdgefarmSysOperatorJWTKey is the key in the edgefarm-sys secret that contains the operator jwt
	AnckEdgefarmSysOperatorJWTKey = "operator-jwt"
	// AnckEdgefarmSysSysAccountCreds is the key in the edgefarm-sys secret that contains the sys account credentials
	AnckEdgefarmSysSysAccountCreds = "sys-creds"
	// AnckEdgefarmSysSysAccountJWT is the key in the edgefarm-sys secret that contains the sys account jwt
	AnckEdgefarmSysSysAccountJWT = "sys-jwt"
	// AnckEdgefarmSysSysAccounPublicKey is the key in the edgefarm-sys secret that contains the sys account public key
	AnckEdgefarmSysSysAccounPublicKey = "sys-public-key"
	// NatsDeniedExportTopics are topics that are denied for exporting
	NatsDeniedExportTopics = "local.>"
)

var (
	// AnckcredentialsServiceURL is the URL of the anck-credentials service
	AnckcredentialsServiceURL = fmt.Sprintf("%s.%s.svc.cluster.local:%d", AnckcredentialsServiceName, AnckNamespace, AnckcredentialsServicePort)
)
