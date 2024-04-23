package main

import "strings"

type stringsvar []string

func (s *stringsvar) String() string {
	return strings.Join(*s, ",")
}

func (s *stringsvar) Set(n string) error {
	*s = append(*s, n)
	return nil
}
