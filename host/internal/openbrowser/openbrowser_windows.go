//go:build windows

package openbrowser

func cmd(url string) (string, []string) {
	return "rundll32", []string{"url.dll,FileProtocolHandler", url}
}
