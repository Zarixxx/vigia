package checks

func Classify(daysLeft, warnDays, critDays int) string {
	switch {
	case daysLeft < 0:
		return "CADUCADO"
	case daysLeft < critDays:
		return "CRÍTICO"
	case daysLeft < warnDays:
		return "AVISO"
	default:
		return "OK"
	}
}
