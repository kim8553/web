package exchangebinding

// ClientPreview is the exchange-form presentation state proven from the
// exact-current form_exchange.lua branch. It is intentionally separate from
// Decision: client preview fields are not, by themselves, authoritative proof
// of the final persisted item BindStatus.
type ClientPreview struct {
	Visible bool
	Bound   bool
}

// PreviewFromRuntime mirrors the exact-current client presentation rules:
//   - BindStatus <= 0: hide the binding preview.
//   - BindStatus > 0 and ShowBind == 0: hide the binding preview.
//   - otherwise show "unbound" unless ExchangeBind > 0, which shows "bound".
//
// Do not use this helper to derive a final Decision. In particular,
// exchangeBind == 0 only proves what the exchange form displays; the complete
// server-side output-binding precedence is still unresolved.
func PreviewFromRuntime(bindStatus, showBind, exchangeBind int32) ClientPreview {
	if bindStatus <= 0 || showBind == 0 {
		return ClientPreview{}
	}
	return ClientPreview{Visible: true, Bound: exchangeBind > 0}
}
