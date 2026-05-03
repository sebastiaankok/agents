package github

import (
	"regexp"
	"strconv"
	"strings"
)

var blockedByRe = regexp.MustCompile(`(?i)blocked by #(\d+)`)

func ParseBlockedBy(body string) []int {
	var result []int
	for _, line := range strings.Split(body, "\n") {
		if m := blockedByRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			result = append(result, n)
		}
	}
	return result
}
