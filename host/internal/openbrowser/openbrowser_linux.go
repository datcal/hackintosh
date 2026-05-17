//go:build linux

package openbrowser

func cmd(url string) (string, []string) {
	return "xdg-open", []string{url}
}
