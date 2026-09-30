package gateway

func serverInstructions() string {
	if text, ok := serverInstructionsByLang[gatewayLang()]; ok {
		return text
	}
	return serverInstructionsByLang["en"]
}
