package cmd

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
)

func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func dim(s string) string    { return colorDim + s + colorReset }
func bold(s string) string   { return colorBold + s + colorReset }
func cyan(s string) string   { return colorCyan + s + colorReset }
