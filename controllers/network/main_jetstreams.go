package network

import (
	"context"
	"fmt"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	jetstreams "github.com/edgefarm/anck/pkg/jetstreams"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	mainJetstreamLog = ctrl.Log.WithName("main-jetstreams")
)

const (
	mainDomain          = "main"
	jsMirrorLinkType    = "mirror"
	jsAggregateLinkType = "aggregate"
)

func (r *NetworksReconciler) deleteMainJetstreams(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	domainMessages := jetstreams.NewDomainMessages()

	mainStreamNames := func(streams []networkv1alpha1.StreamSpec) []string {
		var names []string
		for _, stream := range streams {
			if stream.Location == mainDomain {
				names = append(names, fmt.Sprintf("%s_%s", network.Spec.App, stream.Name))
			}
		}
		return names
	}(network.Spec.Streams)

	anckCreds, err := readCredentialsFromSecret(appComponentName(network.Spec.App, anckParticipant), network.Name, network.Spec.Namespace)
	if err != nil {
		errorText := "Error reading credentials from secret"
		mainJetstreamLog.Error(err, errorText)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	js, err := jetstreams.NewJetstreamController(anckCreds)
	if err != nil {
		errorText := "Error creating jetstream controller"
		mainJetstreamLog.Info(errorText)
		return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
	}
	defer js.Cleanup()

	err = js.Delete(mainDomain, network.Name, mainStreamNames)
	if err != nil {
		errorText := fmt.Sprintf("Error deleting jetstream: {%s: %s, %s: %s, %s: %s}", "domain", mainDomain, "stream", mainStreamNames, "error", err.Error())
		// mainJetstreamLog.Info(errorText)
		domainMessages.Error(mainDomain, errorText)
	} else {
		text := fmt.Sprintf("Successfully deleted jetstream: {%s: %s, %s: %s}", "domain", mainDomain, "stream", mainStreamNames)
		// mainJetstreamLog.Info(text)
		domainMessages.Ok(mainDomain, text)
	}

	for _, msg := range domainMessages.OkMap[mainDomain] {
		mainJetstreamLog.Info(msg)
	}

	if len(domainMessages.ErrMap[mainDomain]) > 0 {
		for _, msg := range domainMessages.ErrMap[mainDomain] {
			mainJetstreamLog.Info(msg)
		}
		return ctrl.Result{}, fmt.Errorf("err: %s", domainMessages.ErrMap[mainDomain])
	}
	return ctrl.Result{}, nil
}

func (r *NetworksReconciler) createMainJetstreams(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	domainMessages := jetstreams.NewDomainMessages()
	networkCopy := network.DeepCopy()

	standard, aggreate := func(streams []networkv1alpha1.StreamSpec) ([]networkv1alpha1.StreamSpec, []networkv1alpha1.StreamSpec) {
		var std []networkv1alpha1.StreamSpec
		var agg []networkv1alpha1.StreamSpec
		for _, stream := range streams {
			if stream.Location == mainDomain {
				if stream.Link != nil {
					agg = append(agg, stream)
					continue
				}
				std = append(std, stream)
			}
		}
		return std, agg
	}(network.Spec.Streams)

	anckCreds, err := readCredentialsFromSecret(appComponentName(network.Spec.App, anckParticipant), network.Name, network.Spec.Namespace)
	if err != nil {
		errorText := "Error reading credentials from secret"
		mainJetstreamLog.Error(err, errorText)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	js, err := jetstreams.NewJetstreamController(anckCreds)
	if err != nil {
		errorText := "Error creating jetstream manager"
		mainJetstreamLog.Info(errorText)
		return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
	}
	defer js.Cleanup()

	// Handle standard streams for main domain
	for _, stream := range standard {
		exists, err := js.Exists(mainDomain, stream.Name)
		if err != nil {
			errorText := fmt.Sprintf("Error checking if jetstream exists: {%s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name)
			domainMessages.Error(mainDomain, errorText)
			break
		}
		if exists {
			domainMessages.Ok(mainDomain, fmt.Sprintf("Jetstream exists: {%s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name))
			continue
		}
		mainJetstreamLog.Info("creating stream", "domain", mainDomain, "name", stream.Name, "network", network.Name)
		err = js.Create(mainDomain, network.Name, stream, network.Spec.Subjects)
		if err != nil {
			errorText := fmt.Sprintf("Error creating jetstream: {%s: %s, %s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name, "error", err.Error())
			domainMessages.Error(mainDomain, errorText)
			network.Info.MainDomain.Standard[stream.Name] = "error"
			break
		}
		domainMessages.Ok(mainDomain, fmt.Sprintf("Successfully created jetstream: {%s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name))
		network.Info.MainDomain.Standard[stream.Name] = "created"
	}

	// Handle aggreate streams for main domain
	for _, stream := range aggreate {
		currentNodes := []string{}
		for node, state := range network.Info.Participating.Nodes {
			if state == "active" {
				currentNodes = append(currentNodes, node)
			}
		}

		err = js.CreateAggregate(mainDomain, network, stream, currentNodes)
		if err != nil {
			errorText := fmt.Sprintf("Error creating mirror jetstream: {%s: %s, %s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name, "error", err.Error())
			domainMessages.Error(mainDomain, errorText)
			network.Info.MainDomain.Aggregatte[stream.Name] = networkv1alpha1.AggreagateStreamSpec{
				SourceDomains: currentNodes,
				SourceName:    stream.Name,
				State:         "error",
			}
			continue
		}
		domainMessages.Ok(mainDomain, fmt.Sprintf("Successfully created mirror jetstream: {%s: %s, %s: %s}", "domain", mainDomain, "stream", stream.Name))
		network.Info.MainDomain.Aggregatte[stream.Name] = networkv1alpha1.AggreagateStreamSpec{
			SourceDomains: currentNodes,
			SourceName:    stream.Name,
			State:         "created",
		}
	}

	err = r.updateInfoAndReturn(ctx, network, networkCopy)
	if err != nil {
		errorText := "Error updating network"
		mainJetstreamLog.Error(err, errorText)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
