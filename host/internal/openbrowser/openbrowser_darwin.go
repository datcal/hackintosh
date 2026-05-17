//go:build darwin

package openbrowser

func cmd(url string) (string, []string) {
	return "open", []string{url}
}
