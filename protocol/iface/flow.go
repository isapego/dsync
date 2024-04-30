package iface

type FlowID struct {
	ID string
}

type FlowOptions struct {
	Type uint

	// for unidirectional flows
	SrcId, DstId        ConnectorID
	SrcConnectorOptions ConnectorOptions
}

const (
	UnidirectionalFlowType = iota
	BidirectionalFlowType
)

type FlowDataIntegrityCheckResult struct {
	Passed bool
}

type FlowStatus struct {
	//For uni-directional flows
	SrcStatus, DstStatus ConnectorStatus
}
