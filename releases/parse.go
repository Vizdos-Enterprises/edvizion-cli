package releases

import "strings"

func ParseRepository(repo string) (owner, name string) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
