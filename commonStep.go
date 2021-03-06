package mig

import (
	"crypto/md5"
	"encoding/base64"
	"strings"
)

func (s *step) computeHash() {
	if s.migrateFunc != nil {
		s.hash = s.migrate
		return
	}
	sum := md5.Sum([]byte(s.migrate))
	b64 := base64.StdEncoding.EncodeToString(sum[:])
	s.hash = string(b64[:])
}

func (s *step) cleanWhitespace() {
	// we want the hash to be invariant to whitespace
	s.migrate = cleanWhitespace(s.migrate)
}

func cleanWhitespace(str string) string {
	var resultLines []string
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		line := strings.TrimSpace(line)

		//skip empty lines
		if len(line) == 0 {
			continue
		}

		//skip comments
		if len(line) >= 2 && line[0:2] == "--" {
			continue
		}

		resultLines = append(resultLines, line)
	}

	return strings.Join(resultLines, "\n")
}
