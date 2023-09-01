package handler

import (
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func parseView(view connectorPB.View) connectorPB.View {
	parsedView := connectorPB.View_VIEW_BASIC
	if view != connectorPB.View_VIEW_UNSPECIFIED {
		parsedView = view
	}
	return parsedView
}
