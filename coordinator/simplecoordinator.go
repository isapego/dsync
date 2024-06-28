package coordinator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/adiom-data/dsync/protocol/iface"
)

type SimpleCoordinator struct {
	// Implement the necessary fields here
	ctx context.Context
	t   iface.Transport
	s   iface.Statestore

	connectors    map[iface.ConnectorID]ConnectorDetailsWithEp
	mu_connectors sync.RWMutex // to make the map thread-safe

	flows    map[iface.FlowID]*FlowDetails
	mu_flows sync.RWMutex // to make the map thread-safe
}

func NewSimpleCoordinator() *SimpleCoordinator {
	// Implement the NewSimpleCoordinator function
	return &SimpleCoordinator{connectors: make(map[iface.ConnectorID]ConnectorDetailsWithEp), flows: make(map[iface.FlowID]*FlowDetails)}
}

// *****
// Thread-safe methods to work with items in the connectors map
// *****

// gets a connector by id
func (c *SimpleCoordinator) getConnector(cid iface.ConnectorID) (ConnectorDetailsWithEp, bool) {
	c.mu_connectors.RLock()
	defer c.mu_connectors.RUnlock()
	connector, ok := c.connectors[cid]
	return connector, ok
}

// deletes a connector from the map
func (c *SimpleCoordinator) delConnector(cid iface.ConnectorID) {
	c.mu_connectors.Lock()
	defer c.mu_connectors.Unlock()
	delete(c.connectors, cid)
}

// adds a connector with a unqiue ID and returns the ID
func (c *SimpleCoordinator) addConnector(connector ConnectorDetailsWithEp) iface.ConnectorID {
	c.mu_connectors.Lock()
	defer c.mu_connectors.Unlock()

	var cid iface.ConnectorID

	for {
		cid = generateConnectorID()
		if _, ok := c.connectors[cid]; !ok {
			break
		}
	}

	connector.Details.Id = cid // set the ID in the details for easier operations later
	c.connectors[cid] = connector

	return cid
}

// return a list of ConnectorDetails for all known connectors
func (c *SimpleCoordinator) GetConnectors() []iface.ConnectorDetails {
	c.mu_connectors.RLock()
	defer c.mu_connectors.RUnlock()
	connectors := make([]iface.ConnectorDetails, 0, len(c.connectors))
	for _, details := range c.connectors {
		connectors = append(connectors, details.Details)
	}
	return connectors
}

// *****
// Thread-safe methods to work with items in the flows map
// *****

// gets a flow by id
func (c *SimpleCoordinator) getFlow(fid iface.FlowID) (*FlowDetails, bool) {
	c.mu_flows.RLock()
	defer c.mu_flows.RUnlock()
	flow, ok := c.flows[fid]
	return flow, ok
}

// adds a flow and returns the ID
func (c *SimpleCoordinator) addFlow(details *FlowDetails) iface.FlowID {
	c.mu_flows.Lock()
	defer c.mu_flows.Unlock()

	var fid iface.FlowID

	for {
		fid = generateFlowID()
		if _, ok := c.flows[fid]; !ok {
			break
		}
	}

	details.FlowID = fid // set the ID in the details for easier operations later
	c.flows[fid] = details

	return fid
}

// removes a flow from the map
func (c *SimpleCoordinator) delFlow(fid iface.FlowID) {
	c.mu_flows.Lock()
	defer c.mu_flows.Unlock()
	delete(c.flows, fid)
}

// *****
// Implement the Coordinator interface methods
// *****

func (c *SimpleCoordinator) Setup(ctx context.Context, t iface.Transport, s iface.Statestore) {
	// Implement the Setup method
	c.ctx = ctx
	c.t = t
	c.s = s
}

func (c *SimpleCoordinator) Teardown() {
	// Implement the Teardown method
}

func (c *SimpleCoordinator) RegisterConnector(details iface.ConnectorDetails, cep iface.ConnectorICoordinatorSignal) (iface.ConnectorID, error) {
	slog.Info("Registering connector with details: " + fmt.Sprintf("%v", details))

	// Check that the details.Id is empty, otherwise error
	if details.Id.ID != "" {
		return iface.ConnectorID{}, fmt.Errorf("we don't support re-registering connectors yet")
	}

	// Add the connector to the list
	cid := c.addConnector(ConnectorDetailsWithEp{Details: details, Endpoint: cep})
	slog.Debug("assigned connector ID: " + cid.ID)

	// Implement the RegisterConnector method
	return cid, nil
}

func (c *SimpleCoordinator) DelistConnector(cid iface.ConnectorID) {
	slog.Info("Deregistering connector with ID: " + fmt.Sprintf("%v", cid))

	// Implement the DelistConnector method
	c.delConnector(cid)
}

func (c *SimpleCoordinator) FlowCreate(o iface.FlowOptions) (iface.FlowID, error) {
	// Check flow type and error out if not unidirectional
	if o.Type != iface.UnidirectionalFlowType {
		return iface.FlowID{}, fmt.Errorf("only unidirectional flows are supported")
	}

	// for unidirectional flows we need two data channels
	// 0 corresponds to the source and 1 to the destination
	// here we're getting away with a trick to short circuit using a single channel
	// TODO (AK, 6/2024): use different channels for source and destination (could be a thing to negotiate between connectors)
	dc0, err := c.t.CreateDataChannel()
	if err != nil {
		return iface.FlowID{}, fmt.Errorf("failed to create data channel 0: %v", err)
	}
	dataChannels := make([]iface.DataChannelID, 2)
	dataChannels[0] = dc0
	dataChannels[1] = dc0

	doneChannels := make([]chan struct{}, 2)
	doneChannels[0] = make(chan struct{})
	doneChannels[1] = make(chan struct{})

	integrityCheckChannels := make([]chan iface.ConnectorDataIntegrityCheckResult, 2)
	integrityCheckChannels[0] = make(chan iface.ConnectorDataIntegrityCheckResult, 1) //XXX: creating buffered channels for now to avoid blocking on writes
	integrityCheckChannels[1] = make(chan iface.ConnectorDataIntegrityCheckResult, 1) //XXX: creating buffered channels for now to avoid blocking on writes

	fdet := FlowDetails{
		Options:                    o,
		DataChannels:               dataChannels,
		DoneNotificationChannels:   doneChannels,
		IntegrityCheckDoneChannels: integrityCheckChannels,
		flowDone:                   make(chan struct{}),
		readPlanningDone:           make(chan struct{}),
	}
	fid := c.addFlow(&fdet)

	slog.Debug("Created flow with ID: " + fmt.Sprintf("%v", fid) + " and options: " + fmt.Sprintf("%v", o))

	return fid, nil
}

func (c *SimpleCoordinator) FlowStart(fid iface.FlowID) error {
	slog.Info("Starting flow with ID: " + fmt.Sprintf("%v", fid))

	// Get the flow details
	flowDet, ok := c.getFlow(fid)
	if !ok {
		return fmt.Errorf("flow %v not found", fid)
	}

	// Get the source and destination connectors
	src, ok := c.getConnector(flowDet.Options.SrcId)
	if !ok {
		return fmt.Errorf("source connector %v not found", flowDet.Options.SrcId)
	}
	dst, ok := c.getConnector(flowDet.Options.DstId)
	if !ok {
		return fmt.Errorf("destination connector %v not found", flowDet.Options.DstId)
	}

	// TODO (AK, 6/2024): Determine shared capabilities and set parameters on src and dst connectors
	// or maybe that should go into the FlowCreate method

	// Request the source connector to create a plan for reading
	if err := src.Endpoint.RequestCreateReadPlan(fid, flowDet.Options.SrcConnectorOptions); err != nil {
		slog.Error("Failed to request read planning from source", err)
		return err
	}

	// Wait for the read planning to be done
	//XXX: we should probably make it async and have a timeout
	select {
	case <-flowDet.readPlanningDone:
		slog.Debug("Read planning done. Flow ID: " + fmt.Sprintf("%v", fid))
	case <-c.ctx.Done():
		slog.Debug("Context cancelled. Flow ID: " + fmt.Sprintf("%v", fid))
		return fmt.Errorf("context cancelled while waiting for read planning to be done")
	}

	// Tell source connector to start reading into the data channel
	if err := src.Endpoint.StartReadToChannel(fid, flowDet.Options.SrcConnectorOptions, flowDet.ReadPlan, flowDet.DataChannels[0]); err != nil {
		slog.Error("Failed to start reading from source", err)
		return err
	}
	// Tell destination connector to start writing from the channel
	if err := dst.Endpoint.StartWriteFromChannel(fid, flowDet.DataChannels[1]); err != nil {
		slog.Error("Failed to start writing to the destination", err)
		return err
	}

	slog.Info("Flow with ID: " + fmt.Sprintf("%v", fid) + " is running")

	go func() {
		// Async wait until both src and dst signal that they are done
		// Exit if the context has been cancelled
		slog.Debug("Waiting for source to finish. Flow ID: " + fmt.Sprintf("%v", fid))
		select {
		case <-flowDet.DoneNotificationChannels[0]:
			slog.Debug("Source finished. Flow ID: " + fmt.Sprintf("%v", fid))
		case <-c.ctx.Done():
			slog.Debug("Context cancelled. Flow ID: " + fmt.Sprintf("%v", fid))
		}
		slog.Debug("Waiting for destination to finish. Flow ID: " + fmt.Sprintf("%v", fid))
		select {
		case <-flowDet.DoneNotificationChannels[1]:
			slog.Debug("Destination finished. Flow ID: " + fmt.Sprintf("%v", fid))
		case <-c.ctx.Done():
			slog.Debug("Context cancelled. Flow ID: " + fmt.Sprintf("%v", fid))
		}
		slog.Info("Flow with ID: " + fmt.Sprintf("%v", fid) + " is done")
		close(flowDet.flowDone)
	}()

	return nil
}

func (c *SimpleCoordinator) WaitForFlowDone(flowId iface.FlowID) error {
	// Get the flow details
	flowDet, ok := c.getFlow(flowId)
	if !ok {
		return fmt.Errorf("flow not found")
	}

	// Wait for the flow to be done
	<-flowDet.flowDone //TODO (AK, 6/2024): should we just return the channel?

	return nil
}

func (c *SimpleCoordinator) FlowStop(fid iface.FlowID) {
	//TODO (AK, 6/2024): Implement the FlowStop method
}

func (c *SimpleCoordinator) FlowDestroy(fid iface.FlowID) {

	slog.Debug("Destroying flow with ID: " + fmt.Sprintf("%v", fid))

	// Get the flow details
	flowDet, err := c.getFlow(fid)
	if !err {
		slog.Error(fmt.Sprintf("Flow %v not found", fid))
	}
	// close the data channels
	for _, ch := range flowDet.DataChannels {
		c.t.CloseDataChannel(ch)
	}

	// close done notification channels - not needed
	// for _, ch := range flowDet.DoneNotificationChannels {
	// 	close(ch)
	// }

	// remove the flow from the map
	c.delFlow(fid)
}

func (c *SimpleCoordinator) NotifyDone(flowId iface.FlowID, conn iface.ConnectorID) error {
	// Get the flow details
	flowDet, ok := c.getFlow(flowId)
	if !ok {
		return fmt.Errorf("flow not found")
	}

	// Check if the connector corresponds to the source
	if flowDet.Options.SrcId == conn {
		// Close the first notification channel
		close(flowDet.DoneNotificationChannels[0])
		return nil
	}

	// Check if the connector corresponds to the destination
	if flowDet.Options.DstId == conn {
		// Close the second notification channel
		close(flowDet.DoneNotificationChannels[1])
		return nil
	}

	return fmt.Errorf("connector not part of the flow")
}

func (c *SimpleCoordinator) PerformFlowIntegrityCheck(fid iface.FlowID) (iface.FlowDataIntegrityCheckResult, error) {
	slog.Info("Initiating flow integrity check for flow with ID: " + fmt.Sprintf("%v", fid))

	res := iface.FlowDataIntegrityCheckResult{}

	// Get the flow details
	flowDet, ok := c.getFlow(fid)
	if !ok {
		return res, fmt.Errorf("flow %v not found", fid)
	}

	// Get the source and destination connectors
	src, ok := c.getConnector(flowDet.Options.SrcId)
	if !ok {
		return res, fmt.Errorf("source connector %v not found", flowDet.Options.SrcId)
	}
	dst, ok := c.getConnector(flowDet.Options.DstId)
	if !ok {
		return res, fmt.Errorf("destination connector %v not found", flowDet.Options.DstId)
	}

	if !src.Details.Cap.IntegrityCheck || !dst.Details.Cap.IntegrityCheck {
		return res, fmt.Errorf("one or both connectors don't support integrity checks")
	}

	// Wait for integrity check results asynchronously
	slog.Debug("Waiting for integrity check results")
	var resSource, resDestination iface.ConnectorDataIntegrityCheckResult

	// Request integrity check results from connectors
	if err := src.Endpoint.RequestDataIntegrityCheck(fid, flowDet.Options.SrcConnectorOptions); err != nil {
		slog.Error("Failed to request integrity check from source", err)
		return res, err
	}
	if err := dst.Endpoint.RequestDataIntegrityCheck(fid, iface.ConnectorOptions{}); err != nil { //TODO (AK, 6/2024): should we have proper options here? (maybe even data validation-specific?)
		slog.Error("Failed to request integrity check from destination", err)
		return res, err
	}

	// Wait for both results
	select {
	case resSource = <-flowDet.IntegrityCheckDoneChannels[0]:
		slog.Debug("Got integrity check result from source: " + fmt.Sprintf("%v", resSource))
	case <-c.ctx.Done():
		slog.Debug("Context cancelled. Flow ID: " + fmt.Sprintf("%v", fid))
	}
	select {
	case resDestination = <-flowDet.IntegrityCheckDoneChannels[1]:
		slog.Debug("Got integrity check result from destination: " + fmt.Sprintf("%v", resDestination))
	case <-c.ctx.Done():
		slog.Debug("Context cancelled. Flow ID: " + fmt.Sprintf("%v", fid))
	}

	if (resSource == iface.ConnectorDataIntegrityCheckResult{}) || (resDestination == iface.ConnectorDataIntegrityCheckResult{}) {
		slog.Debug("Integrity check results are empty")
		return res, fmt.Errorf("integrity check results are empty")
	}

	if (!resSource.Success) || (!resDestination.Success) {
		slog.Debug("Integrity check failure on either end")
		return res, fmt.Errorf("integrity check failure on either end")
	}

	if resSource != resDestination {
		slog.Debug("Checksums don't match")
		res.Passed = false
	} else {
		slog.Debug("Checksums match")
		res.Passed = true
	}

	return res, nil
}

func (c *SimpleCoordinator) PostDataIntegrityCheckResult(flowId iface.FlowID, conn iface.ConnectorID, res iface.ConnectorDataIntegrityCheckResult) error {
	// Get the flow details
	flowDet, ok := c.getFlow(flowId)
	if !ok {
		return fmt.Errorf("flow not found")
	}

	// Check if the connector corresponds to the source
	if flowDet.Options.SrcId == conn {
		flowDet.IntegrityCheckDoneChannels[0] <- res //post the result to the channel
		close(flowDet.DoneNotificationChannels[0])   //close the notification channel to indicate that we're done here //XXX: not sure if we need this
		return nil
	}

	// Check if the connector corresponds to the destination
	if flowDet.Options.DstId == conn {
		flowDet.IntegrityCheckDoneChannels[1] <- res //post the result to the channel
		close(flowDet.DoneNotificationChannels[1])   //close the notification channel to indicate that we're done here //XXX: not sure if we need this
		return nil
	}

	return fmt.Errorf("connector not part of the flow")
}

func (c *SimpleCoordinator) GetFlowStatus(fid iface.FlowID) (iface.FlowStatus, error) {
	res := iface.FlowStatus{}

	// Get the flow details
	flowDet, ok := c.getFlow(fid)
	if !ok {
		return res, fmt.Errorf("flow %v not found", fid)
	}

	// Get the source and destination connectors
	src, ok := c.getConnector(flowDet.Options.SrcId)
	if !ok {
		return res, fmt.Errorf("source connector %v not found", flowDet.Options.SrcId)
	}
	dst, ok := c.getConnector(flowDet.Options.DstId)
	if !ok {
		return res, fmt.Errorf("destination connector %v not found", flowDet.Options.DstId)
	}

	// Get latest status update from connectors
	flowDet.flowStatus.SrcStatus = src.Endpoint.GetConnectorStatus(fid)
	flowDet.flowStatus.DstStatus = dst.Endpoint.GetConnectorStatus(fid)

	return flowDet.flowStatus, nil
}

func (c *SimpleCoordinator) UpdateConnectorStatus(flowId iface.FlowID, conn iface.ConnectorID, status iface.ConnectorStatus) error {
	// Get the flow details
	flowDet, ok := c.getFlow(flowId)
	if !ok {
		return fmt.Errorf("flow not found")
	}

	// Check if the connector corresponds to the source
	if flowDet.Options.SrcId == conn {
		flowDet.flowStatus.SrcStatus = status
		return nil
	}

	// Check if the connector corresponds to the destination
	if flowDet.Options.DstId == conn {
		flowDet.flowStatus.DstStatus = status
		return nil
	}

	return fmt.Errorf("connector not part of the flow")
}

func (c *SimpleCoordinator) PostReadPlanningResult(flowId iface.FlowID, conn iface.ConnectorID, res iface.ConnectorReadPlanResult) error {
	slog.Debug(fmt.Sprintf("Got read plan result from connector %v for flow %v", conn, flowId))
	// Get the flow details
	flowDet, ok := c.getFlow(flowId)
	if !ok {
		return fmt.Errorf("flow not found")
	}

	//sanity check that the connector is the source
	if flowDet.Options.SrcId != conn {
		return fmt.Errorf("connector not the source for the flow")
	}

	//check that the result was a success
	if !res.Success {
		return fmt.Errorf("read planning failed")
	}

	flowDet.ReadPlan = res.ReadPlan
	close(flowDet.readPlanningDone) //close the channel to indicate that we got the plan
	return nil
}
